package brand

import (
	"smart-wardrobe-be/internal/modules/brand/application"
	"smart-wardrobe-be/internal/modules/brand/infrastructure"
	"smart-wardrobe-be/internal/modules/brand/presentation/contract"
	"smart-wardrobe-be/internal/modules/brand/presentation/handler"
	"smart-wardrobe-be/internal/modules/brand/presentation/worker"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	infrastructure.ProviderSet,
	application.ProviderSet,
	contract.NewBrandContract,
	contract.NewBrandDashboardReader,
	handler.NewBrandPortalHandler,
	handler.NewBrandLoyaltyHandler,
	handler.NewBrandChatHandler,
	handler.NewBrandHandler,
	worker.NewLoyaltyPointExpiryWorker,
)
