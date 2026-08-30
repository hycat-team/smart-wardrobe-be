package wardrobe

import (
	outfit_uc "smart-wardrobe-be/internal/modules/wardrobe/application/usecase/outfit"
	"smart-wardrobe-be/internal/modules/wardrobe/application/usecase/wardrobe"
	"smart-wardrobe-be/internal/modules/wardrobe/application/usecase/wardrobe/catalog"
	"smart-wardrobe-be/internal/modules/wardrobe/application/usecase/wardrobe/contractuc"
	"smart-wardrobe-be/internal/modules/wardrobe/application/usecase/wardrobe/item"
	"smart-wardrobe-be/internal/modules/wardrobe/application/usecase/wardrobe/search_sync"
	"smart-wardrobe-be/internal/modules/wardrobe/infrastructure/persistence"
	"smart-wardrobe-be/internal/modules/wardrobe/infrastructure/search"
	"smart-wardrobe-be/internal/modules/wardrobe/presentation/contract"
	"smart-wardrobe-be/internal/modules/wardrobe/presentation/handler"
	"smart-wardrobe-be/internal/modules/wardrobe/presentation/router"
	presentation_worker "smart-wardrobe-be/internal/modules/wardrobe/presentation/worker"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	persistence.NewWardrobeItemRepository,
	persistence.NewOutfitRepository,
	search.NewWardrobeSearchService,
	search.NewWardrobeSearchIndexService,

	item.NewWardrobeItemUseCase,
	catalog.NewWardrobeCatalogUseCase,
	contractuc.NewWardrobeContractUseCase,
	contract.NewWardrobeContract,
	contract.NewWardrobeDashboardReader,
	search_sync.NewSearchSyncUseCase,
	wardrobe.NewWardrobeFashionEventUseCase,

	outfit_uc.NewOutfitUseCase,
	handler.NewWardrobeItemHandler,
	handler.NewOutfitHandler,
	router.NewRouter,
	presentation_worker.NewSearchSyncWorker,
	presentation_worker.NewWardrobeFashionEventWorker,
	presentation_worker.NewWardrobeCleanupWorker,
	presentation_worker.NewWardrobeProcessingExhaustedWorker,
	presentation_worker.NewWardrobeFashionBindWorker,
	presentation_worker.NewElasticsearchProcessingExhaustedWorker,
)
