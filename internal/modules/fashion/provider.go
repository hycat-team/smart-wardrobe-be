package fashion

import (
	"smart-wardrobe-be/internal/modules/fashion/application/usecase/ai/chat"
	"smart-wardrobe-be/internal/modules/fashion/application/usecase/ai/recommendation"
	"smart-wardrobe-be/internal/modules/fashion/application/usecase/worker"
	"smart-wardrobe-be/internal/modules/fashion/contract"
	"smart-wardrobe-be/internal/modules/fashion/infrastructure/persistence"
	"smart-wardrobe-be/internal/modules/fashion/presentation/handler"
	presentation_worker "smart-wardrobe-be/internal/modules/fashion/presentation/worker"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	persistence.NewFashionItemRepository,
	persistence.NewConversationalContextRepository,
	persistence.NewMessageRepository,

	chat.NewWardrobeChatUseCase,
	recommendation.NewOutfitRecommendationUseCase,
	worker.NewVisionCategoryCache,
	worker.NewWardrobeWorkerUseCase,

	contract.NewFashionContract,

	handler.NewWardrobeAIHandler,
	presentation_worker.NewFailedItemsCleanupWorker,
	presentation_worker.NewProcessingRecoveryWorker,
	presentation_worker.NewWardrobeBatchUploadWorker,
	presentation_worker.NewFashionAnalyzeWorker,
)
