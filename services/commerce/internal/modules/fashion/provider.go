package fashion

import (
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/usecase/ai/chat"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/usecase/ai/recommendation"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/usecase/category"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/usecase/fashion_item"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/usecase/worker"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/infrastructure/persistence"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/presentation/contract"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/presentation/handler"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/presentation/router"
	presentation_worker "smart-wardrobe-be/services/commerce/internal/modules/fashion/presentation/worker"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	persistence.NewFashionItemRepository,
	persistence.NewCategoryRepository,
	persistence.NewConversationalContextRepository,
	persistence.NewMessageRepository,
	// 033 T021: personalization persistence. The style learning worker and the
	// style profile use case (later phases) are their consumers.
	persistence.NewUserStyleProfileRepository,
	persistence.NewStyleSignalRepository,

	chat.NewWardrobeChatUseCase,
	recommendation.NewOutfitRecommendationUseCase,
	worker.NewVisionCategoryCache,
	worker.NewFashionWorkerUseCase,
	worker.NewFashionInitUseCase,

	fashion_item.NewFashionItemUseCase,
	category.NewCategoryUseCase,
	contract.NewFashionContract,
	contract.NewCategoryContract,
	contract.NewFashionDashboardReader,

	handler.NewWardrobeAIHandler,
	handler.NewCategoryHandler,
	router.NewRouter,
	presentation_worker.NewFailedItemsCleanupWorker,
	presentation_worker.NewProcessingRecoveryWorker,
	presentation_worker.NewFashionInitWorker,
	presentation_worker.NewFashionAnalyzeWorker,
)
