package wardrobe

import (
	"context"
	"fmt"

	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/modules/subscription/contract"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/application/interface/search"
	uc_interfaces "smart-wardrobe-be/internal/modules/wardrobe/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/repositories"
	"smart-wardrobe-be/internal/shared/application/ai"
	"smart-wardrobe-be/internal/shared/application/constants/apperror"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	"smart-wardrobe-be/internal/shared/application/event"
	"smart-wardrobe-be/internal/shared/application/media"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"
	"smart-wardrobe-be/pkg/logger"

	"smart-wardrobe-be/internal/modules/wardrobe/application/mapper"

	"github.com/google/uuid"
)

type WardrobeUseCase struct {
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
	return &WardrobeUseCase{
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
}

func (uc *WardrobeUseCase) GetUploadSignature(ctx context.Context) (*shared_dto.UploadSignatureResult, error) {
	folder := uc.cfg.Cloudinary.ItemFolder
	if folder == "" {
		folder = "smart_wardrobe/items"
	}
	params := shared_dto.UploadSignatureParams{
		Folder: folder,
	}
	return uc.mediaService.GenerateUploadSignature(ctx, params)
}

func (uc *WardrobeUseCase) GetWardrobeItems(ctx context.Context, userID uuid.UUID) ([]*dto.WardrobeItemRes, error) {
	subOverview, err := uc.userSubContract.GetUserSubscriptionOverview(ctx, userID)
	if err != nil {
		return nil, err
	}

	items, err := uc.wardrobeRepo.GetByUserID(ctx, userID)
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

func (uc *WardrobeUseCase) GetWardrobeItemByID(ctx context.Context, userID uuid.UUID, id uuid.UUID) (*dto.WardrobeItemRes, error) {
	item, err := uc.wardrobeRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if item == nil || item.UserID != userID {
		return nil, apperror.NewNotFound("Không tìm thấy trang phục này.")
	}

	subOverview, err := uc.userSubContract.GetUserSubscriptionOverview(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Read active items list to determine lock status by order
	items, err := uc.wardrobeRepo.GetByUserID(ctx, userID)
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
		return nil, apperror.NewForbidden(fmt.Sprintf("Trang phục này đã bị khóa vì tủ đồ vượt quá giới hạn tối đa của gói dịch vụ (Tối đa %d món).", subOverview.MaxWardrobeItems))
	}

	res := mapper.MapToWardrobeItemRes(item)
	res.IsLocked = false
	return res, nil
}

