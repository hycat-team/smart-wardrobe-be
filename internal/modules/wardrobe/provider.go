package wardrobe

import (
	wardrobe_uc "smart-wardrobe-be/internal/modules/wardrobe/application/usecase/wardrobe"
	outfit_uc "smart-wardrobe-be/internal/modules/wardrobe/application/usecase/outfit"
	"smart-wardrobe-be/internal/modules/wardrobe/infrastructure/persistence"
	"smart-wardrobe-be/internal/modules/wardrobe/presentation/handler"
	"smart-wardrobe-be/internal/modules/wardrobe/presentation/worker"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	persistence.NewWardrobeItemRepository,
	persistence.NewCategoryRepository,
	persistence.NewOutfitRepository,
	wardrobe_uc.NewWardrobeUseCase,
	outfit_uc.NewOutfitUseCase,
	handler.NewWardrobeHandler,
	handler.NewOutfitHandler,
	worker.NewBatchCropRabbitMQWorker,
	worker.NewElasticsearchSyncWorker,
)
