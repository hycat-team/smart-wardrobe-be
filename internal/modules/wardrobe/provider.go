package wardrobe

import (
	wardrobe_uc "smart-wardrobe-be/internal/modules/wardrobe/application/usecase/wardrobe"
	"smart-wardrobe-be/internal/modules/wardrobe/infrastructure/persistence"
	"smart-wardrobe-be/internal/modules/wardrobe/presentation/handler"
	"smart-wardrobe-be/internal/modules/wardrobe/presentation/worker"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	persistence.NewWardrobeItemRepository,
	persistence.NewCategoryRepository,
	wardrobe_uc.NewWardrobeUseCase,
	handler.NewWardrobeHandler,
	worker.NewBatchCropRabbitMQWorker,
)
