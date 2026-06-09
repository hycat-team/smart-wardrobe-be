package wardrobe

import (
	"context"

	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/modules/subscription/contract"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	wardrobeerrors "smart-wardrobe-be/internal/modules/wardrobe/application/errors"
	"smart-wardrobe-be/internal/modules/wardrobe/application/interface/search"
	uc_interfaces "smart-wardrobe-be/internal/modules/wardrobe/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/repositories"
	"smart-wardrobe-be/internal/shared/application/ai"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	"smart-wardrobe-be/internal/shared/application/event"
	"smart-wardrobe-be/internal/shared/application/media"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"
	"smart-wardrobe-be/pkg/logger"

	"smart-wardrobe-be/internal/modules/wardrobe/application/mapper"

	"github.com/google/uuid"
)

type wardrobeUseCaseSupport struct {
	cfg             *config.Config
	logger          logger.Interface
	wardrobeRepo    repositories.IWardrobeItemRepository
	categoryRepo    repositories.ICategoryRepository
	contextRepo     repositories.IConversationalContextRepository
	messageRepo     repositories.IMessageRepository
	searchEngine    search.IWardrobeSearchService
	mediaService    media.IMediaService
	aiService       ai.IAIService
	userSubContract contract.IUserSubscriptionContract
	userQuotaCtr    contract.IUserQuotaContract
	eventPublisher  event.IEventPublisher
	uow             shared_repos.IUnitOfWork
}

type WardrobeItemUseCase struct {
	*wardrobeUseCaseSupport
}

type WardrobeAIUseCase struct {
	*wardrobeUseCaseSupport
}

type WardrobeCatalogUseCase struct {
	*wardrobeUseCaseSupport
}

type WardrobeWorkerUseCase struct {
	*wardrobeUseCaseSupport
}

type WardrobeContractUseCase struct {
	*wardrobeUseCaseSupport
}

type WardrobeUseCase struct {
	*WardrobeItemUseCase
	*WardrobeAIUseCase
	*WardrobeCatalogUseCase
	*WardrobeWorkerUseCase
	*WardrobeContractUseCase
}

func NewWardrobeUseCase(
	cfg *config.Config,
	l logger.Interface,
	wardrobeRepo repositories.IWardrobeItemRepository,
	categoryRepo repositories.ICategoryRepository,
	contextRepo repositories.IConversationalContextRepository,
	messageRepo repositories.IMessageRepository,
	searchEngine search.IWardrobeSearchService,
	mediaService media.IMediaService,
	aiService ai.IAIService,
	userSubContract contract.IUserSubscriptionContract,
	userQuotaCtr contract.IUserQuotaContract,
	eventPublisher event.IEventPublisher,
	uow shared_repos.IUnitOfWork,
) uc_interfaces.IWardrobeUseCase {
	support := &wardrobeUseCaseSupport{
		cfg:             cfg,
		logger:          l,
		wardrobeRepo:    wardrobeRepo,
		categoryRepo:    categoryRepo,
		contextRepo:     contextRepo,
		messageRepo:     messageRepo,
		searchEngine:    searchEngine,
		mediaService:    mediaService,
		aiService:       aiService,
		userSubContract: userSubContract,
		userQuotaCtr:    userQuotaCtr,
		eventPublisher:  eventPublisher,
		uow:             uow,
	}

	return &WardrobeUseCase{
		WardrobeItemUseCase:     &WardrobeItemUseCase{wardrobeUseCaseSupport: support},
		WardrobeAIUseCase:       &WardrobeAIUseCase{wardrobeUseCaseSupport: support},
		WardrobeCatalogUseCase:  &WardrobeCatalogUseCase{wardrobeUseCaseSupport: support},
		WardrobeWorkerUseCase:   &WardrobeWorkerUseCase{wardrobeUseCaseSupport: support},
		WardrobeContractUseCase: &WardrobeContractUseCase{wardrobeUseCaseSupport: support},
	}
}

func (uc *WardrobeItemUseCase) GetUploadSignature(ctx context.Context) (*shared_dto.UploadSignatureResult, error) {
	folder := uc.cfg.Cloudinary.ItemFolder
	if folder == "" {
		folder = "smart_wardrobe/items"
	}
	params := shared_dto.UploadSignatureParams{
		Folder: folder,
	}
	return uc.mediaService.GenerateUploadSignature(ctx, params)
}

func (uc *WardrobeItemUseCase) GetWardrobeItems(ctx context.Context, userID uuid.UUID, query dto.GetWardrobeItemsQueryReq) ([]*dto.WardrobeItemRes, error) {
	subOverview, err := uc.userSubContract.GetUserSubscriptionOverview(ctx, userID)
	if err != nil {
		return nil, err
	}

	var categorySlug *string
	if query.CategorySlug != "" {
		categorySlug = &query.CategorySlug
	}

	items, err := uc.wardrobeRepo.GetByUserID(ctx, userID, categorySlug)
	if err != nil {
		return nil, err
	}

	resList := make([]*dto.WardrobeItemRes, len(items))
	for idx, item := range items {
		res := mapper.MapToWardrobeItemRes(item)
		if idx >= subOverview.MaxWardrobeItems {
			res.IsLocked = true
		} else {
			res.IsLocked = false
		}
		resList[idx] = res
	}

	return resList, nil
}

func (uc *WardrobeItemUseCase) GetWardrobeItemByID(ctx context.Context, userID uuid.UUID, id uuid.UUID) (*dto.WardrobeItemRes, error) {
	item, err := uc.wardrobeRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if item == nil || item.UserID != userID {
		return nil, wardrobeerrors.ErrItemNotFound
	}

	subOverview, err := uc.userSubContract.GetUserSubscriptionOverview(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Read active items list to determine lock status by order
	items, err := uc.wardrobeRepo.GetByUserID(ctx, userID, nil)
	if err != nil {
		return nil, err
	}

	isLocked := false
	for idx, it := range items {
		if it.ID == id {
			if idx >= subOverview.MaxWardrobeItems {
				isLocked = true
			}
			break
		}
	}

	if isLocked {
		return nil, wardrobeerrors.ErrItemLockedDueToLimit(subOverview.MaxWardrobeItems)
	}

	res := mapper.MapToWardrobeItemRes(item)
	res.IsLocked = false
	return res, nil
}
