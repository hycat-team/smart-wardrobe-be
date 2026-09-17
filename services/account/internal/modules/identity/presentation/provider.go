package presentation

import (
	"smart-wardrobe-be/services/account/contract/identity"
	"smart-wardrobe-be/services/account/internal/modules/identity/presentation/handler"
	"smart-wardrobe-be/services/account/internal/modules/identity/presentation/router"
	shared_middleware "smart-wardrobe-be/shared/presentation/middleware"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	handler.NewAuthHandler,
	handler.NewAdminHandler,
	handler.NewMeHandler,
	contract.NewUserDashboardReader,
	shared_middleware.NewAuthMiddleware,
	wire.Bind(new(contract.IAuthMiddleware), new(*shared_middleware.AuthMiddleware)),
	shared_middleware.NewLoginRateLimitMiddleware,
	router.NewRouter,
)
