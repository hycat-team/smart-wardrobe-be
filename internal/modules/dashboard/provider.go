package dashboard

import (
	"smart-wardrobe-be/internal/modules/dashboard/application/usecase"
	"smart-wardrobe-be/internal/modules/dashboard/infrastructure/persistence"
	"smart-wardrobe-be/internal/modules/dashboard/presentation/handler"
	"smart-wardrobe-be/internal/modules/dashboard/presentation/worker"

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
	worker.NewDashboardAggregatorWorker,
)
