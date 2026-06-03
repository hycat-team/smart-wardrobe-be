package wardrobe

import (
	category_uc "smart-wardrobe-be/internal/modules/wardrobe/application/usecase/category"
	outfit_uc "smart-wardrobe-be/internal/modules/wardrobe/application/usecase/outfit"
	wardrobe_uc "smart-wardrobe-be/internal/modules/wardrobe/application/usecase/wardrobe"
	"smart-wardrobe-be/internal/modules/wardrobe/infrastructure/messaging"
	"smart-wardrobe-be/internal/modules/wardrobe/infrastructure/persistence"
	"smart-wardrobe-be/internal/modules/wardrobe/infrastructure/search"
	"smart-wardrobe-be/internal/modules/wardrobe/presentation/handler"
	"smart-wardrobe-be/internal/modules/wardrobe/presentation/worker"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	persistence.NewWardrobeItemRepository,
	persistence.NewCategoryRepository,
	persistence.NewOutfitRepository,
	search.NewWardrobeSearchService,
	search.NewWardrobeSearchIndexService,
	messaging.NewWardrobeBatchUploadJobConsumer,
	messaging.NewSearchSyncEventConsumer,
	wardrobe_uc.NewWardrobeUseCase,
	outfit_uc.NewOutfitUseCase,
	category_uc.NewCategoryUseCase,
	handler.NewWardrobeHandler,
	handler.NewOutfitHandler,
	handler.NewCategoryHandler,
	worker.NewWardrobeBatchUploadWorker,
	worker.NewSearchSyncWorker,
	worker.NewFailedItemsCleanupWorker,
)
