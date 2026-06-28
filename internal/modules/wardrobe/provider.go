package wardrobe

import (
	category_uc "smart-wardrobe-be/internal/modules/wardrobe/application/usecase/category"
	outfit_uc "smart-wardrobe-be/internal/modules/wardrobe/application/usecase/outfit"
	"smart-wardrobe-be/internal/modules/wardrobe/application/usecase/wardrobe/catalog"
	"smart-wardrobe-be/internal/modules/wardrobe/application/usecase/wardrobe/contractuc"
	"smart-wardrobe-be/internal/modules/wardrobe/application/usecase/wardrobe/item"
	"smart-wardrobe-be/internal/modules/wardrobe/application/usecase/wardrobe/search_sync"
	"smart-wardrobe-be/internal/modules/wardrobe/infrastructure/messaging"
	"smart-wardrobe-be/internal/modules/wardrobe/infrastructure/persistence"
	"smart-wardrobe-be/internal/modules/wardrobe/infrastructure/search"
	"smart-wardrobe-be/internal/modules/wardrobe/presentation/handler"
	presentation_worker "smart-wardrobe-be/internal/modules/wardrobe/presentation/worker"

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

	item.NewWardrobeItemUseCase,
	catalog.NewWardrobeCatalogUseCase,
	contractuc.NewWardrobeContractUseCase,
	search_sync.NewSearchSyncUseCase,

	outfit_uc.NewOutfitUseCase,
	category_uc.NewCategoryUseCase,
	handler.NewWardrobeItemHandler,
	handler.NewOutfitHandler,
	handler.NewCategoryHandler,
	presentation_worker.NewSearchSyncWorker,
)
