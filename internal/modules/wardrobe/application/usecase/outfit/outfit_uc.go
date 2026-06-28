package outfit

import (
	"context"
	"time"

	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/modules/subscription/contract"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	wardrobeerrors "smart-wardrobe-be/internal/modules/wardrobe/application/errors"
	uc_interfaces "smart-wardrobe-be/internal/modules/wardrobe/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/wardrobe/application/mapper"
	"smart-wardrobe-be/internal/modules/wardrobe/application/usecase/wardrobe/shared"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/repositories"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	"smart-wardrobe-be/internal/shared/application/media"
	"smart-wardrobe-be/internal/shared/domain/constants/outfititemcontext"
	"smart-wardrobe-be/internal/shared/domain/constants/outfitstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/wardrobestatus"
	"smart-wardrobe-be/internal/shared/domain/entities"
	"smart-wardrobe-be/pkg/logger"

	"github.com/google/uuid"
)

type OutfitUseCase struct {
	cfg             *config.Config
	logger          logger.Interface
	outfitRepo      repositories.IOutfitRepository
	wardrobeRepo    repositories.IWardrobeItemRepository
	userSubContract contract.IUserSubscriptionContract
	mediaService    media.IMediaService
}

func NewOutfitUseCase(
	cfg *config.Config,
	l logger.Interface,
	outfitRepo repositories.IOutfitRepository,
	wardrobeRepo repositories.IWardrobeItemRepository,
	userSubContract contract.IUserSubscriptionContract,
	mediaService media.IMediaService,
) uc_interfaces.IOutfitUseCase {
	return &OutfitUseCase{
		cfg:             cfg,
		logger:          l,
		outfitRepo:      outfitRepo,
		wardrobeRepo:    wardrobeRepo,
		userSubContract: userSubContract,
		mediaService:    mediaService,
	}
}

func (uc *OutfitUseCase) GetUploadSignature(ctx context.Context) (*shared_dto.UploadSignatureResult, error) {
	folder := uc.cfg.Cloudinary.OutfitFolder
	if folder == "" {
		folder = "smart_wardrobe/outfits"
	}

	return uc.mediaService.GenerateUploadSignature(ctx, shared_dto.UploadSignatureParams{
		Folder: folder,
	})
}

func (uc *OutfitUseCase) SaveOutfit(ctx context.Context, userID uuid.UUID, input dto.SaveOutfitReq) (*dto.OutfitRes, error) {
	// 1. Check outfit limit of the subscription plan
	subOverview, err := uc.userSubContract.GetUserSubscriptionOverview(ctx, userID)
	if err != nil {
		return nil, err
	}

	existingOutfits, err := uc.outfitRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if len(existingOutfits) >= subOverview.MaxOutfits {
		return nil, wardrobeerrors.ErrOutfitLimitReached(len(existingOutfits), subOverview.MaxOutfits)
	}

	// 2. Resolve requested fashion items to concrete user wardrobe items.
	fashionItemIDs := make([]uuid.UUID, len(input.Items))
	for idx, itemReq := range input.Items {
		fashionItemIDs[idx] = itemReq.FashionItemID
	}

	verifiedItems, err := uc.wardrobeRepo.GetByUserIDAndFashionItemIDs(ctx, userID, fashionItemIDs)
	if err != nil {
		return nil, err
	}

	verifiedMap := make(map[uuid.UUID]*entities.WardrobeItem)
	for _, item := range verifiedItems {
		if item.Status == wardrobestatus.Sold {
			return nil, wardrobeerrors.ErrOutfitItemSold(item.FashionItemID)
		}
		verifiedMap[item.FashionItemID] = item
	}

	userItems, err := uc.wardrobeRepo.GetByUserID(ctx, userID, nil)
	if err != nil {
		return nil, err
	}
	lockedMap := shared.BuildLockedMap(userItems, subOverview.MaxWardrobeItems)

	// Verify that all items are valid
	outfitItems := make([]*entities.OutfitItem, len(input.Items))
	wardrobeItemIDs := make([]uuid.UUID, 0, len(input.Items))
	for idx, itemReq := range input.Items {
		verifiedItem, ok := verifiedMap[itemReq.FashionItemID]
		if !ok || verifiedItem.FashionItemID == uuid.Nil {
			return nil, wardrobeerrors.ErrOutfitItemInvalidOrForbidden(itemReq.FashionItemID)
		}

		if lockedMap[verifiedItem.ID] {
			return nil, wardrobeerrors.ErrItemLockedDueToLimit(subOverview.MaxWardrobeItems)
		}
		wardrobeItemIDs = append(wardrobeItemIDs, verifiedItem.ID)

		outfitItems[idx] = &entities.OutfitItem{
			FashionItemID: verifiedItem.FashionItemID,
			FashionItem:   verifiedItem.FashionItem,
			ItemContext:   outfititemcontext.UserWardrobe,
			WardrobeItem:  verifiedItem,
			PositionX:     itemReq.PositionX,
			PositionY:     itemReq.PositionY,
			Scale:         itemReq.Scale,
			LayerOrder:    itemReq.LayerOrder,
		}
	}

	// 3. Perform Transaction to create Outfit
	outfit := &entities.Outfit{
		UserID:        userID,
		Name:          input.Name,
		Description:   input.Description,
		CoverImageUrl: input.CoverImageUrl,
		CoverPublicID: input.CoverPublicID,
		Status:        outfitstatus.Active,
	}

	err = uc.outfitRepo.CreateWithItems(ctx, outfit, outfitItems)
	if err != nil {
		return nil, err
	}

	if err := uc.touchLastUsedAt(ctx, wardrobeItemIDs); err != nil {
		uc.logger.Warn("Failed to update wardrobe last_used_at after saving outfit")
	}

	return mapper.MapToOutfitRes(outfit, outfitItems), nil
}

func (uc *OutfitUseCase) UpdateOutfit(ctx context.Context, userID uuid.UUID, id uuid.UUID, input dto.SaveOutfitReq) (*dto.OutfitRes, error) {
	subOverview, err := uc.userSubContract.GetUserSubscriptionOverview(ctx, userID)
	if err != nil {
		return nil, err
	}

	// 1. Check if the Outfit exists and belongs to the User
	outfit, err := uc.outfitRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if outfit == nil || outfit.UserID != userID {
		return nil, wardrobeerrors.ErrOutfitNotFound()
	}

	// 2. Resolve requested fashion items to concrete user wardrobe items.
	fashionItemIDs := make([]uuid.UUID, len(input.Items))
	for idx, itemReq := range input.Items {
		fashionItemIDs[idx] = itemReq.FashionItemID
	}

	verifiedItems, err := uc.wardrobeRepo.GetByUserIDAndFashionItemIDs(ctx, userID, fashionItemIDs)
	if err != nil {
		return nil, err
	}

	verifiedMap := make(map[uuid.UUID]*entities.WardrobeItem)
	for _, item := range verifiedItems {
		if item.Status == wardrobestatus.Sold {
			return nil, wardrobeerrors.ErrOutfitItemSold(item.FashionItemID)
		}
		verifiedMap[item.FashionItemID] = item
	}

	userItems, err := uc.wardrobeRepo.GetByUserID(ctx, userID, nil)
	if err != nil {
		return nil, err
	}
	lockedMap := shared.BuildLockedMap(userItems, subOverview.MaxWardrobeItems)

	outfitItems := make([]*entities.OutfitItem, len(input.Items))
	wardrobeItemIDs := make([]uuid.UUID, 0, len(input.Items))
	for idx, itemReq := range input.Items {
		verifiedItem, ok := verifiedMap[itemReq.FashionItemID]
		if !ok || verifiedItem.FashionItemID == uuid.Nil {
			return nil, wardrobeerrors.ErrOutfitItemInvalidOrForbidden(itemReq.FashionItemID)
		}

		if lockedMap[verifiedItem.ID] {
			return nil, wardrobeerrors.ErrItemLockedDueToLimit(subOverview.MaxWardrobeItems)
		}
		wardrobeItemIDs = append(wardrobeItemIDs, verifiedItem.ID)

		outfitItems[idx] = &entities.OutfitItem{
			OutfitID:      id,
			FashionItemID: verifiedItem.FashionItemID,
			FashionItem:   verifiedItem.FashionItem,
			ItemContext:   outfititemcontext.UserWardrobe,
			WardrobeItem:  verifiedItem,
			PositionX:     itemReq.PositionX,
			PositionY:     itemReq.PositionY,
			Scale:         itemReq.Scale,
			LayerOrder:    itemReq.LayerOrder,
		}
	}

	// 3. Perform Update
	outfit.Name = input.Name
	outfit.Description = input.Description
	outfit.CoverImageUrl = input.CoverImageUrl
	outfit.CoverPublicID = input.CoverPublicID

	err = uc.outfitRepo.UpdateWithItems(ctx, outfit, outfitItems)
	if err != nil {
		return nil, err
	}

	if err := uc.touchLastUsedAt(ctx, wardrobeItemIDs); err != nil {
		uc.logger.Warn("Failed to update wardrobe last_used_at after updating outfit")
	}

	return mapper.MapToOutfitRes(outfit, outfitItems), nil
}

func (uc *OutfitUseCase) GetOutfits(ctx context.Context, userID uuid.UUID, query dto.GetOutfitsQueryReq) (*shared_dto.PaginationResult[*dto.OutfitRes], error) {
	page := query.Page
	if page <= 0 {
		page = 1
	}
	limit := query.Limit
	if limit <= 0 {
		limit = 20
	}

	paginationQuery := shared_dto.PaginationQuery{
		Page:  page,
		Limit: limit,
	}

	totalItems, err := uc.outfitRepo.CountByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	outfits, err := uc.outfitRepo.GetByUserIDPaginated(ctx, userID, paginationQuery)
	if err != nil {
		return nil, err
	}

	resList := make([]*dto.OutfitRes, len(outfits))
	for idx, outfit := range outfits {
		resList[idx] = mapper.MapToOutfitRes(outfit, nil)
	}

	return &shared_dto.PaginationResult[*dto.OutfitRes]{
		Items:    resList,
		Metadata: shared_dto.BuildPaginationMetadata(paginationQuery, totalItems),
	}, nil
}

func (uc *OutfitUseCase) GetOutfitByID(ctx context.Context, userID uuid.UUID, id uuid.UUID) (*dto.OutfitRes, error) {
	outfit, items, err := uc.outfitRepo.GetDetailByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if outfit == nil || outfit.UserID != userID {
		return nil, wardrobeerrors.ErrOutfitNotFound()
	}

	return mapper.MapToOutfitRes(outfit, items), nil
}

func (uc *OutfitUseCase) DeleteOutfit(ctx context.Context, userID uuid.UUID, id uuid.UUID) error {
	outfit, err := uc.outfitRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if outfit == nil || outfit.UserID != userID {
		return wardrobeerrors.ErrOutfitNotFound()
	}

	return uc.outfitRepo.DeleteOutfit(ctx, id)
}

func (uc *OutfitUseCase) touchLastUsedAt(ctx context.Context, itemIDs []uuid.UUID) error {
	now := time.Now().UTC()
	return uc.wardrobeRepo.TouchLastUsedAt(ctx, itemIDs, now)
}
