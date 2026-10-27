package fashion

import (
	usecase_interfaces "smart-wardrobe-be/services/commerce/internal/modules/fashion/application/interface/usecase"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/usecase/ai/chat"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/usecase/ai/recommendation"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/usecase/category"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/usecase/fashion_item"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/usecase/personalization/completion"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/usecase/personalization/feedback"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/usecase/personalization/learning"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/usecase/personalization/profile"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/usecase/worker"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/infrastructure/caching"
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

	// 033 T038: Redis caches for catalog suggestions and recommendation
	// snapshots. Both implement fail-soft application ports.
	caching.NewRedisPersonalizationCatalogCache,
	caching.NewRedisRecommendationSnapshotCache,

	chat.NewWardrobeChatUseCase,
	recommendation.NewOutfitRecommendationUseCase,
	// 033 T077: the recommendation use case is the shared catalog suggestion
	// builder of US2 and US7, exposed through two interfaces from one instance.
	wire.Bind(new(usecase_interfaces.IOutfitRecommendationUseCase), new(*recommendation.OutfitRecommendationUseCase)),
	wire.Bind(new(usecase_interfaces.IOutfitSuggestionBuilder), new(*recommendation.OutfitRecommendationUseCase)),
	// 033 T077: outfit completion validates ownership and reuses that builder.
	completion.NewOutfitCompletionUseCase,
	worker.NewVisionCategoryCache,
	worker.NewFashionWorkerUseCase,
	worker.NewFashionInitUseCase,

	fashion_item.NewFashionItemUseCase,
	category.NewCategoryUseCase,
	// 033 T027/T028: style profile read/update/delete (US1).
	profile.NewStyleProfileUseCase,
	// 033 T060: style learning shared by the outfit-saved worker (US4) and the
	// feedback endpoint (US6).
	learning.NewStyleLearningUseCase,
	// 033 T072: recommendation feedback (US6) reads the snapshot and reuses the
	// shared learning use case.
	feedback.NewRecommendationFeedbackUseCase,
	contract.NewFashionContract,
	contract.NewCategoryContract,
	contract.NewFashionDashboardReader,

	handler.NewWardrobeAIHandler,
	handler.NewCategoryHandler,
	handler.NewStyleProfileHandler,
	router.NewRouter,
	presentation_worker.NewFailedItemsCleanupWorker,
	presentation_worker.NewProcessingRecoveryWorker,
	presentation_worker.NewFashionInitWorker,
	presentation_worker.NewFashionAnalyzeWorker,
	// 033 T061/T062: wardrobe.outfit.saved -> style_learning_queue -> learned
	// style profile.
	presentation_worker.NewStyleLearningWorker,
)
