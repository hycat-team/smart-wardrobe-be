package brand

import (
	"smart-wardrobe-be/services/commerce/internal/modules/brand/application"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/infrastructure"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/presentation/contract"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/presentation/handler"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/presentation/middleware"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/presentation/router"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/presentation/worker"

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
	handler.NewAdminBrandHandler,
	handler.NewBrandHandler,
	middleware.NewBrandRoleMiddleware,
	wire.Bind(new(contract.IBrandRoleMiddleware), new(*middleware.BrandRoleMiddleware)),
	middleware.NewBrandRefMiddleware,
	wire.Bind(new(contract.IBrandRefMiddleware), new(*middleware.BrandRefMiddleware)),
	router.NewRouter,
	worker.NewLoyaltyPointExpiryWorker,
	worker.NewBrandFashionEventWorker,
	worker.NewBrandFashionBindWorker,
	worker.NewBrandProcessingExhaustedWorker,
)
