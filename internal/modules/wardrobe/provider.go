package wardrobe

import (
	uc_interfaces "smart-wardrobe-be/internal/modules/wardrobe/application/interface/usecase"
	category_uc "smart-wardrobe-be/internal/modules/wardrobe/application/usecase/category"
	outfit_uc "smart-wardrobe-be/internal/modules/wardrobe/application/usecase/outfit"
	wardrobe_uc "smart-wardrobe-be/internal/modules/wardrobe/application/usecase/wardrobe"
	"smart-wardrobe-be/internal/modules/wardrobe/contract"
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
	persistence.NewConversationalContextRepository,
	persistence.NewMessageRepository,
	search.NewWardrobeSearchService,
	search.NewWardrobeSearchIndexService,
	messaging.NewWardrobeBatchUploadJobConsumer,
	messaging.NewSearchSyncEventConsumer,
	wardrobe_uc.NewWardrobeUseCase,
	wire.Bind(new(contract.IWardrobeContract), new(uc_interfaces.IWardrobeUseCase)),
	outfit_uc.NewOutfitUseCase,
	category_uc.NewCategoryUseCase,
	handler.NewWardrobeHandler,
	handler.NewOutfitHandler,
	handler.NewCategoryHandler,
	worker.NewWardrobeBatchUploadWorker,
	worker.NewSearchSyncWorker,
	worker.NewFailedItemsCleanupWorker,
)
