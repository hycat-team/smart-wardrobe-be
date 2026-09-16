package worker

import (
	"smart-wardrobe-be/services/account/contract/subscription"
	uc_interfaces "smart-wardrobe-be/services/commerce/internal/modules/fashion/application/interface/usecase"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/domain/repositories"
	wardrobe_contract "smart-wardrobe-be/services/commerce/internal/modules/wardrobe/presentation/contract"
	"smart-wardrobe-be/shared/config"

	aiexecutor_contract "smart-wardrobe-be/services/ai/contract/aiexecutor"
	"smart-wardrobe-be/shared/application/event"
	"smart-wardrobe-be/shared/application/media"
	shared_repos "smart-wardrobe-be/shared/domain/repositories"
	"smart-wardrobe-be/shared/pkg/logger"
)

const (
	processingFailureReasonMessage = "Hệ thống chưa thể phân tích trang phục lúc này. Vui lòng thử lại sau."
	autoRetryExceededMessage       = "Đã vượt quá số lần thử xử lý tự động."
)

// FashionWorkerUseCase orchestrates asynchronous wardrobe item processing and recovery workflows.
type FashionWorkerUseCase struct {
	cfg              *config.Config
	logger           logger.Interface
	wardrobeContract wardrobe_contract.IWardrobeContract
	categoryCache    *VisionCategoryCache
	mediaService     media.IMediaService
	aiService        aiexecutor_contract.IAIOrchestrator
	userSubContract  contract.IUserSubscriptionContract
	eventPublisher   event.IEventPublisher
	fashionItemRepo  repositories.IFashionItemRepository
	uow              shared_repos.IUnitOfWork
}

// NewFashionWorkerUseCase creates the worker use case responsible for AI item processing jobs.
func NewFashionWorkerUseCase(
	cfg *config.Config,
	logger logger.Interface,
	wardrobeContract wardrobe_contract.IWardrobeContract,
	categoryCache *VisionCategoryCache,
	mediaService media.IMediaService,
	aiService aiexecutor_contract.IAIOrchestrator,
	userSubContract contract.IUserSubscriptionContract,
	eventPublisher event.IEventPublisher,
	fashionItemRepo repositories.IFashionItemRepository,
	uow shared_repos.IUnitOfWork,
) uc_interfaces.IFashionWorkerUseCase {
	return &FashionWorkerUseCase{
		cfg:              cfg,
		logger:           logger,
		wardrobeContract: wardrobeContract,
		categoryCache:    categoryCache,
		mediaService:     mediaService,
		aiService:        aiService,
		userSubContract:  userSubContract,
		eventPublisher:   eventPublisher,
		fashionItemRepo:  fashionItemRepo,
		uow:              uow,
	}
}

// ProcessBackgroundBatchUploadJob analyzes a processing wardrobe item and finalizes its AI metadata.
