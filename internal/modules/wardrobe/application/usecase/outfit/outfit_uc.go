package outfit

import (
	"context"

	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/modules/subscription/contract"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	wardrobeerrors "smart-wardrobe-be/internal/modules/wardrobe/application/errors"
	uc_interfaces "smart-wardrobe-be/internal/modules/wardrobe/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/wardrobe/application/mapper"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/repositories"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	"smart-wardrobe-be/internal/shared/application/media"
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

	// 2. Check if the provided wardrobe items exist and belong to the user
	itemIDs := make([]uuid.UUID, len(input.Items))
	for idx, itemReq := range input.Items {
		itemIDs[idx] = itemReq.WardrobeItemID
	}

	verifiedItems, err := uc.wardrobeRepo.GetByIDs(ctx, itemIDs)
	if err != nil {
		return nil, err
	}

	verifiedMap := make(map[uuid.UUID]*entities.WardrobeItem)
	for _, item := range verifiedItems {
		if item.UserID == userID {
			if item.Status == wardrobestatus.Sold {
				return nil, wardrobeerrors.ErrOutfitItemSold(item.ID)
			}
			verifiedMap[item.ID] = item
		}
	}

	// Verify that all items are valid
	outfitItems := make([]*entities.OutfitItem, len(input.Items))
	for idx, itemReq := range input.Items {
		verifiedItem, ok := verifiedMap[itemReq.WardrobeItemID]
		if !ok {
			return nil, wardrobeerrors.ErrOutfitItemInvalidOrForbidden(itemReq.WardrobeItemID)
		}

		outfitItems[idx] = &entities.OutfitItem{
			ItemID:     itemReq.WardrobeItemID,
			Wardrobe:   verifiedItem,
			PositionX:  itemReq.PositionX,
			PositionY:  itemReq.PositionY,
			Scale:      itemReq.Scale,
			LayerOrder: itemReq.LayerOrder,
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

	return mapper.MapToOutfitRes(outfit, outfitItems), nil
}

func (uc *OutfitUseCase) UpdateOutfit(ctx context.Context, userID uuid.UUID, id uuid.UUID, input dto.SaveOutfitReq) (*dto.OutfitRes, error) {
	// 1. Check if the Outfit exists and belongs to the User
	outfit, err := uc.outfitRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if outfit == nil || outfit.UserID != userID {
		return nil, wardrobeerrors.ErrOutfitNotFound
	}

	// 2. Check the provided wardrobe items
	itemIDs := make([]uuid.UUID, len(input.Items))
	for idx, itemReq := range input.Items {
		itemIDs[idx] = itemReq.WardrobeItemID
	}

	verifiedItems, err := uc.wardrobeRepo.GetByIDs(ctx, itemIDs)
	if err != nil {
		return nil, err
	}

	verifiedMap := make(map[uuid.UUID]*entities.WardrobeItem)
	for _, item := range verifiedItems {
		if item.UserID == userID {
			if item.Status == wardrobestatus.Sold {
				return nil, wardrobeerrors.ErrOutfitItemSold(item.ID)
			}
			verifiedMap[item.ID] = item
		}
	}

	outfitItems := make([]*entities.OutfitItem, len(input.Items))
	for idx, itemReq := range input.Items {
		verifiedItem, ok := verifiedMap[itemReq.WardrobeItemID]
		if !ok {
			return nil, wardrobeerrors.ErrOutfitItemInvalidOrForbidden(itemReq.WardrobeItemID)
		}

		outfitItems[idx] = &entities.OutfitItem{
			OutfitID:   id,
			ItemID:     itemReq.WardrobeItemID,
			Wardrobe:   verifiedItem,
			PositionX:  itemReq.PositionX,
			PositionY:  itemReq.PositionY,
			Scale:      itemReq.Scale,
			LayerOrder: itemReq.LayerOrder,
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

	return mapper.MapToOutfitRes(outfit, outfitItems), nil
}

func (uc *OutfitUseCase) GetOutfits(ctx context.Context, userID uuid.UUID) ([]*dto.OutfitRes, error) {
	outfits, err := uc.outfitRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	resList := make([]*dto.OutfitRes, len(outfits))
	for idx, outfit := range outfits {
		resList[idx] = mapper.MapToOutfitRes(outfit, nil)
	}

	return resList, nil
}

func (uc *OutfitUseCase) GetOutfitByID(ctx context.Context, userID uuid.UUID, id uuid.UUID) (*dto.OutfitRes, error) {
	outfit, items, err := uc.outfitRepo.GetDetailByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if outfit == nil || outfit.UserID != userID {
		return nil, wardrobeerrors.ErrOutfitNotFound
	}

	return mapper.MapToOutfitRes(outfit, items), nil
}

func (uc *OutfitUseCase) DeleteOutfit(ctx context.Context, userID uuid.UUID, id uuid.UUID) error {
	outfit, err := uc.outfitRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if outfit == nil || outfit.UserID != userID {
		return wardrobeerrors.ErrOutfitNotFound
	}

	return uc.outfitRepo.DeleteOutfit(ctx, id)
}

