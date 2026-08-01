package presentation

import (
	"smart-wardrobe-be/internal/modules/identity/presentation/contract"
	"smart-wardrobe-be/internal/modules/identity/presentation/handler"
	"smart-wardrobe-be/internal/modules/identity/presentation/middleware"
	"smart-wardrobe-be/internal/modules/identity/presentation/router"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	handler.NewAuthHandler,
	handler.NewAdminHandler,
	handler.NewMeHandler,
	contract.NewUserDashboardReader,
	middleware.NewAuthMiddleware,
	wire.Bind(new(contract.IAuthMiddleware), new(*middleware.AuthMiddleware)),
	router.NewRouter,
)
