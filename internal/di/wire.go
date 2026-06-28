//go:build wireinject
// +build wireinject

package di

import (
	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/api/middleware"
	"smart-wardrobe-be/internal/api/routes"
	"smart-wardrobe-be/internal/bootstrap"
	"smart-wardrobe-be/internal/modules/brand"
	"smart-wardrobe-be/internal/modules/fashion"
	"smart-wardrobe-be/internal/modules/identity"
	"smart-wardrobe-be/internal/modules/subscription"
	"smart-wardrobe-be/internal/modules/wardrobe"
	"smart-wardrobe-be/internal/shared"
	"smart-wardrobe-be/internal/shared/infrastructure/caching"
	"smart-wardrobe-be/internal/shared/infrastructure/db"
	"smart-wardrobe-be/pkg/logger"

	"github.com/google/wire"
)

func InitializeApp(cfg *config.Config, l logger.Interface) (*bootstrap.App, func(), error) {
	wire.Build(
		bootstrap.NewApp,
		db.NewPostgresConnection,
		db.NewGormUnitOfWork,
		caching.NewRedisConnection,

		shared.ProviderSet,
		identity.ProviderSet,
		subscription.ProviderSet,
		wardrobe.ProviderSet,
		brand.ProviderSet,
		fashion.ProviderSet,

		middleware.NewAuthMiddleware,
		middleware.NewRateLimitMiddleware,
		routes.RouterSet,
		routes.NewEngine,
		wire.Struct(new(routes.AppRouter), "*"),
		wire.Struct(new(bootstrap.AppWorkers), "*"),
	)
	return &bootstrap.App{}, nil, nil
}
