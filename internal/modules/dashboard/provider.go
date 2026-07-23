package dashboard

import (
	"smart-wardrobe-be/internal/modules/dashboard/application/usecase"
	"smart-wardrobe-be/internal/modules/dashboard/infrastructure/persistence"
	"smart-wardrobe-be/internal/modules/dashboard/presentation/handler"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	persistence.NewDashboardRepository,
	usecase.NewAdminDashboardUseCase,
	usecase.NewBrandDashboardUseCase,
	usecase.NewUserDashboardUseCase,
	usecase.NewDynamicQueryUseCase,
	usecase.NewDrillDownUseCase,
	handler.NewAdminDashboardHandler,
	handler.NewBrandDashboardHandler,
	handler.NewUserDashboardHandler,
	handler.NewAnalyticsHandler,
)
