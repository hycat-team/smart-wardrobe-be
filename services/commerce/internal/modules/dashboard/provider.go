package dashboard

import (
	"smart-wardrobe-be/services/commerce/internal/modules/dashboard/application/usecase"
	"smart-wardrobe-be/services/commerce/internal/modules/dashboard/infrastructure/persistence"
	"smart-wardrobe-be/services/commerce/internal/modules/dashboard/presentation/handler"
	"smart-wardrobe-be/services/commerce/internal/modules/dashboard/presentation/router"
	"smart-wardrobe-be/services/commerce/internal/modules/dashboard/presentation/worker"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	persistence.NewDashboardRepository,
	usecase.NewDashboardAggregatorUseCase,
	usecase.NewAdminDashboardUseCase,
	usecase.NewBrandDashboardUseCase,
	usecase.NewUserDashboardUseCase,
	usecase.NewDynamicQueryUseCase,
	usecase.NewDrillDownUseCase,
	handler.NewAdminDashboardHandler,
	handler.NewBrandDashboardHandler,
	handler.NewUserDashboardHandler,
	handler.NewAnalyticsHandler,
	router.NewRouter,
	worker.NewDashboardAggregatorWorker,
)
