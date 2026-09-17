//go:build wireinject
// +build wireinject

package di

import (
	servicebootstrap "smart-wardrobe-be/services/account/internal/bootstrap"
	adminmodule "smart-wardrobe-be/services/account/internal/modules/admin"
	admingrpc "smart-wardrobe-be/services/account/internal/modules/admin/presentation/grpc"
	identitymodule "smart-wardrobe-be/services/account/internal/modules/identity"
	identitygrpc "smart-wardrobe-be/services/account/internal/modules/identity/presentation/grpc"
	notificationmodule "smart-wardrobe-be/services/account/internal/modules/notification"
	subscriptionmodule "smart-wardrobe-be/services/account/internal/modules/subscription"
	subscriptiongrpc "smart-wardrobe-be/services/account/internal/modules/subscription/presentation/grpc"
	sharedevent "smart-wardrobe-be/shared/application/event"
	sharedusecase "smart-wardrobe-be/shared/application/interface/usecase"
	sharedUsecase "smart-wardrobe-be/shared/application/usecase"
	sharedconfig "smart-wardrobe-be/shared/config"
	"smart-wardrobe-be/shared/infrastructure/caching"
	infra_db "smart-wardrobe-be/shared/infrastructure/db"
	"smart-wardrobe-be/shared/infrastructure/media"
	"smart-wardrobe-be/shared/infrastructure/messaging"
	"smart-wardrobe-be/shared/infrastructure/search"
	sharedlogger "smart-wardrobe-be/shared/pkg/logger"
	shared_middleware "smart-wardrobe-be/shared/presentation/middleware"
	sharedWorker "smart-wardrobe-be/shared/presentation/worker"

	"github.com/google/wire"
)

// InitializeApp composes the standalone Account service runtime (T034a/T040):
// shared connections, module providers, gRPC servers, HTTP engine and workers,
// selected at runtime by APP_MODE.
func InitializeApp(cfg *sharedconfig.Config, l sharedlogger.Interface) (*servicebootstrap.App, func(), error) {
	wire.Build(
		// Shared infrastructure singletons.
		infra_db.NewPostgresConnection,
		caching.NewRedisConnection,
		caching.NewRedisCacheService,
		caching.NewRedisCooldownService,
		caching.NewRedisRateLimiter,
		media.NewCloudinaryService,
		messaging.NewRabbitMQClient,
		messaging.NewOutboxRepository,
		search.NewElasticsearchClient,
		messaging.NewOutboxGaugeProbe,
		sharedUsecase.NewOutboxUseCase,
		infra_db.NewOutboxListener,
		sharedWorker.NewOutboxWorker,
		wire.Bind(new(sharedusecase.IOutboxUseCase), new(*sharedUsecase.OutboxUseCase)),
		wire.Bind(new(sharedWorker.OutboxSignalListener), new(*infra_db.OutboxListener)),
		wire.Bind(new(sharedevent.IOutboxGaugeProbe), new(*messaging.OutboxGaugeProbe)),
		wire.Bind(new(sharedevent.IRawMessagePublisher), new(*messaging.RabbitMQClient)),
		wire.Bind(new(sharedevent.IMessageConsumer), new(*messaging.RabbitMQClient)),
		wire.Bind(new(sharedevent.IEventPublisher), new(*messaging.RabbitMQClient)),
		infra_db.NewGormUnitOfWork,
		infra_db.NewPostgresJobLocker,

		// Account modules (identity, subscription, admin, notification).
		identitymodule.ProviderSet,
		notificationmodule.ProviderSet,
		subscriptionmodule.ProviderSet,
		adminmodule.ProviderSet,

		// gRPC servers hosted by Account.
		identitygrpc.NewIdentityServer,
		identitygrpc.NewIdentityDashboardServer,
		subscriptiongrpc.NewSubscriptionServer,
		subscriptiongrpc.NewSubscriptionDashboardServer,
		admingrpc.NewAdminServer,

		// HTTP engine + middleware.
		shared_middleware.NewRateLimitMiddleware,
		shared_middleware.NewBodyLimitMiddleware,
		servicebootstrap.NewEngine,

		// Runtime assembly.
		provideMigrationsFS,
		servicebootstrap.NewGRPCServer,
		wire.Struct(new(servicebootstrap.Workers), "*"),
		servicebootstrap.NewApp,
	)
	return &servicebootstrap.App{}, nil, nil
}
