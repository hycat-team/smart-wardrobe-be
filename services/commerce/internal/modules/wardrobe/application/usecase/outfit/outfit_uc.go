package outfit

import (
	"context"
	"time"

	"smart-wardrobe-be/services/account/contract/subscription"
	brand_contract "smart-wardrobe-be/services/commerce/internal/modules/brand/presentation/contract"
	fashion_dto "smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	fashion_contract "smart-wardrobe-be/services/commerce/internal/modules/fashion/presentation/contract"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/dto"
	wardrobeerrors "smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/errors"
	uc_interfaces "smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/interface/usecase"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/mapper"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/usecase/wardrobe/shared"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/constants/outfititemcontext"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/constants/outfitstatus"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/entities"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/repositories"
	"smart-wardrobe-be/shared/application/constants/eventconstants"
	shared_dto "smart-wardrobe-be/shared/application/dto"
	shared_event "smart-wardrobe-be/shared/application/event"
	"smart-wardrobe-be/shared/application/media"
	"smart-wardrobe-be/shared/config"
	shared_repos "smart-wardrobe-be/shared/domain/repositories"
	"smart-wardrobe-be/shared/pkg/logger"
	"smart-wardrobe-be/shared/pkg/utils/sliceutils"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type OutfitUseCase struct {
	cfg             *config.Config
	logger          logger.Interface
	outfitRepo      repositories.IOutfitRepository
	wardrobeRepo    repositories.IWardrobeItemRepository
	userSubContract contract.IUserSubscriptionContract
	brandContract   brand_contract.IBrandContract
	fashionContract fashion_contract.IFashionContract
	mediaService    media.IMediaService
	eventPublisher  shared_event.IEventPublisher
	uow             shared_repos.IUnitOfWork
}

func NewOutfitUseCase(
	cfg *config.Config,
	l logger.Interface,
	outfitRepo repositories.IOutfitRepository,
	wardrobeRepo repositories.IWardrobeItemRepository,
	userSubContract contract.IUserSubscriptionContract,
	brandContract brand_contract.IBrandContract,
	fashionContract fashion_contract.IFashionContract,
	mediaService media.IMediaService,
	eventPublisher shared_event.IEventPublisher,
	uow shared_repos.IUnitOfWork,
) uc_interfaces.IOutfitUseCase {
	return &OutfitUseCase{
		cfg:             cfg,
		logger:          l,
		outfitRepo:      outfitRepo,
		wardrobeRepo:    wardrobeRepo,
		userSubContract: userSubContract,
		brandContract:   brandContract,
		fashionContract: fashionContract,
		mediaService:    mediaService,
		eventPublisher:  eventPublisher,
		uow:             uow,
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
	// 1. Reject a payload that references the same fashion item twice (FR-033).
	if err := ensureUniqueOutfitItems(input.Items); err != nil {
		return nil, err
	}

	// 2. Check outfit limit of the subscription plan.
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

	// 3. Resolve each item to a wardrobe row, brand item or catalog clone
	// (explicit itemContext restricts the source, omitted auto-resolves).
	userItems, err := uc.wardrobeRepo.GetByUserID(ctx, userID, nil)
	if err != nil {
		return nil, err
	}
	lockedMap := shared.BuildLockedMap(userItems, subOverview.MaxWardrobeItems)

	resolution, err := uc.resolveOutfitItems(ctx, userID, uuid.Nil, input.Items, subOverview.MaxWardrobeItems, lockedMap)
	if err != nil {
		return nil, err
	}

	// 4. Outfit + clone + touch + outbox signal commit atomically (D6/D7).
	outfit := &entities.Outfit{
		UserID:        userID,
		Name:          input.Name,
		Description:   input.Description,
		CoverImageUrl: input.CoverImageUrl,
		CoverPublicID: input.CoverPublicID,
		Status:        outfitstatus.Active,
	}

	now := time.Now().UTC()
	err = uc.executeInUnitOfWork(ctx, func(txCtx context.Context) error {
		if err := uc.enforceOutfitCloneLimit(txCtx, userID, resolution, subOverview.MaxWardrobeItems); err != nil {
			return err
		}

		if err := uc.outfitRepo.CreateWithItems(txCtx, outfit, resolution.items); err != nil {
			return err
		}
		if len(resolution.cloneItems) > 0 {
			if err := uc.wardrobeRepo.BulkCreate(txCtx, resolution.cloneItems); err != nil {
				return err
			}
		}
		if err := uc.wardrobeRepo.TouchLastUsedAt(txCtx, resolution.touchIDs(), now); err != nil {
			return err
		}
		return uc.publishOutfitSaved(txCtx, outfit.ID, userID, resolution.allFashionItemIDs())
	})
	if err != nil {
		return nil, err
	}

	return uc.mapOutfitResponse(ctx, outfit, resolution.items), nil
}

func (uc *OutfitUseCase) UpdateOutfit(ctx context.Context, userID uuid.UUID, id uuid.UUID, input dto.SaveOutfitReq) (*dto.OutfitRes, error) {
	if err := ensureUniqueOutfitItems(input.Items); err != nil {
		return nil, err
	}

	subOverview, err := uc.userSubContract.GetUserSubscriptionOverview(ctx, userID)
	if err != nil {
		return nil, err
	}

	// 1. Check if the Outfit exists and belongs to the User (items preloaded).
	outfit, err := uc.outfitRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if outfit == nil || outfit.UserID != userID {
		return nil, wardrobeerrors.ErrOutfitNotFound()
	}

	// 2. Resolve requested items with the same rules as SaveOutfit (FR-034).
	userItems, err := uc.wardrobeRepo.GetByUserID(ctx, userID, nil)
	if err != nil {
		return nil, err
	}
	lockedMap := shared.BuildLockedMap(userItems, subOverview.MaxWardrobeItems)

	resolution, err := uc.resolveOutfitItems(ctx, userID, id, input.Items, subOverview.MaxWardrobeItems, lockedMap)
	if err != nil {
		return nil, err
	}

	// 3. Diff the item set: only newly added items are touched and signalled
	// (FR-034/FR-062).
	previousFashionIDs := fashionItemIDsOf(outfit.Items)
	nextFashionIDs := resolution.allFashionItemIDs()
	addedFashionIDs := newlyAddedFashionItemIDs(previousFashionIDs, nextFashionIDs)
	addedSet := make(map[uuid.UUID]struct{}, len(addedFashionIDs))
	for _, fashionID := range addedFashionIDs {
		addedSet[fashionID] = struct{}{}
	}

	// Preserve the existing outfit status; only the editable metadata changes.
	outfit.Name = input.Name
	outfit.Description = input.Description
	outfit.CoverImageUrl = input.CoverImageUrl
	outfit.CoverPublicID = input.CoverPublicID

	now := time.Now().UTC()
	err = uc.executeInUnitOfWork(ctx, func(txCtx context.Context) error {
		if err := uc.enforceOutfitCloneLimit(txCtx, userID, resolution, subOverview.MaxWardrobeItems); err != nil {
			return err
		}

		if err := uc.outfitRepo.UpdateWithItems(txCtx, outfit, resolution.items); err != nil {
			return err
		}
		if len(resolution.cloneItems) > 0 {
			if err := uc.wardrobeRepo.BulkCreate(txCtx, resolution.cloneItems); err != nil {
				return err
			}
		}
		if err := uc.wardrobeRepo.TouchLastUsedAt(txCtx, resolution.touchIDsFor(addedSet), now); err != nil {
			return err
		}

		// No new items => no learning signal. Only the newly added fashion
		// items are published when the set did change.
		if len(addedFashionIDs) == 0 {
			return nil
		}
		return uc.publishOutfitSaved(txCtx, outfit.ID, userID, addedFashionIDs)
	})
	if err != nil {
		return nil, err
	}

	return uc.mapOutfitResponse(ctx, outfit, resolution.items), nil
}

func (uc *OutfitUseCase) GetOutfits(ctx context.Context, userID uuid.UUID, query dto.GetOutfitsQueryReq) (*shared_dto.PaginationResult[*dto.OutfitBriefRes], error) {
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

	resList := make([]*dto.OutfitBriefRes, len(outfits))
	for idx, outfit := range outfits {
		resList[idx] = mapper.MapToOutfitBrief(outfit)
	}

	return &shared_dto.PaginationResult[*dto.OutfitBriefRes]{
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

	fashionIDs := make([]uuid.UUID, 0, len(items))
	for _, oi := range items {
		fashionIDs = append(fashionIDs, oi.FashionItemID)
	}
	fashionIDs = sliceutils.UniqueUUIDs(fashionIDs)
	fashionMap := map[uuid.UUID]*fashion_dto.FashionItemBriefRes{}
	if len(fashionIDs) > 0 {
		fashionItems, err := uc.fashionContract.ListFashionItemsBrief(ctx, fashionIDs)
		if err == nil {
			fashionMap = sliceutils.IndexBy(fashionItems, func(f *fashion_dto.FashionItemBriefRes) uuid.UUID { return f.ID })
		}
	}

	userFashionIDs := make([]uuid.UUID, 0, len(items))
	brandFashionIDs := make([]uuid.UUID, 0, len(items))
	for _, oi := range items {
		if oi.ItemContext == outfititemcontext.BrandItem {
			brandFashionIDs = append(brandFashionIDs, oi.FashionItemID)
		} else {
			userFashionIDs = append(userFashionIDs, oi.FashionItemID)
		}
	}

	availableFashionIDs := map[uuid.UUID]struct{}{}
	if len(userFashionIDs) > 0 {
		wardrobeItems, err := uc.wardrobeRepo.GetByUserIDAndFashionItemIDs(ctx, userID, userFashionIDs)
		if err == nil {
			for _, wi := range wardrobeItems {
				if wi.FashionItemID != nil {
					availableFashionIDs[*wi.FashionItemID] = struct{}{}
				}
			}
		}
	}
	if len(brandFashionIDs) > 0 {
		activeBrandItems, err := uc.brandContract.ListActiveBrandItemFashionIDs(ctx, brandFashionIDs)
		if err == nil {
			for fid, active := range activeBrandItems {
				if active {
					availableFashionIDs[fid] = struct{}{}
				}
			}
		}
	}

	return mapper.MapToOutfitRes(outfit, items, fashionMap, availableFashionIDs), nil
}

func (uc *OutfitUseCase) DeleteOutfit(ctx context.Context, userID uuid.UUID, id uuid.UUID) error {
	outfit, err := uc.outfitRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if outfit == nil || outfit.UserID != userID {
		return wardrobeerrors.ErrOutfitNotFound()
	}

	// Delete + cascade event are committed atomically: the outbox record is
	// written in the same transaction, so a publish failure rolls the delete
	// back (FR-005/FR-047).
	if uc.uow == nil {
		if err := uc.outfitRepo.DeleteOutfit(ctx, id); err != nil {
			return err
		}
		return uc.publishOutfitDeleted(ctx, id, userID)
	}
	return uc.uow.Execute(ctx, func(txCtx context.Context) error {
		if err := uc.outfitRepo.DeleteOutfit(txCtx, id); err != nil {
			return err
		}
		return uc.publishOutfitDeleted(txCtx, id, userID)
	})
}

func (uc *OutfitUseCase) publishOutfitDeleted(ctx context.Context, outfitID, userID uuid.UUID) error {
	if uc.eventPublisher == nil {
		return nil
	}
	payload := dto.OutfitEventPayload{OutfitID: outfitID, UserID: userID}
	if err := uc.eventPublisher.Publish(ctx, eventconstants.TopicOutfitDeleted, payload); err != nil {
		uc.logger.Error("failed to publish outfit deleted event", zap.String("outfit_id", outfitID.String()), zap.Error(err))
		return err
	}
	return nil
}

// enforceOutfitCloneLimit rejects the whole save/update when cloning the new
// catalog rows would exceed the wardrobe quota (FR-032). It runs inside the
// transaction and takes the per-user wardrobe lock first, so a concurrent save,
// update or closet initialization of the same user cannot pass the check while
// this one is still uncommitted.
func (uc *OutfitUseCase) enforceOutfitCloneLimit(ctx context.Context, userID uuid.UUID, resolution *outfitResolution, maxWardrobeItems int) error {
	if resolution == nil || len(resolution.cloneItems) == 0 {
		return nil
	}
	if err := uc.wardrobeRepo.LockUserWardrobe(ctx, userID); err != nil {
		return err
	}
	currentCount, err := uc.wardrobeRepo.CountByUserID(ctx, userID)
	if err != nil {
		return err
	}
	if exceedsWardrobeLimit(int(currentCount), len(resolution.cloneItems), maxWardrobeItems) {
		return wardrobeerrors.ErrWardrobeLimitExceededForOutfit(resolution.cloneFashionItemIDs())
	}
	return nil
}

// mapOutfitResponse keeps the existing response shape: items are always shown
// as outfit items keyed by fashion item, even when the wardrobe row was cloned.
func (uc *OutfitUseCase) mapOutfitResponse(ctx context.Context, outfit *entities.Outfit, items []*entities.OutfitItem) *dto.OutfitRes {
	fashionIDs := fashionItemIDsOf(items)
	fashionMap := map[uuid.UUID]*fashion_dto.FashionItemBriefRes{}
	if len(fashionIDs) > 0 {
		fashionItems, err := uc.fashionContract.ListFashionItemsBrief(ctx, fashionIDs)
		if err == nil {
			fashionMap = sliceutils.IndexBy(fashionItems, func(f *fashion_dto.FashionItemBriefRes) uuid.UUID { return f.ID })
		}
	}
	return mapper.MapToOutfitRes(outfit, items, fashionMap, nil)
}
