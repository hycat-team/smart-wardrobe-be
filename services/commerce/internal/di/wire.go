//go:build wireinject
// +build wireinject

package di

import (
	identitycontract "smart-wardrobe-be/services/account/contract/identity"
	servicebootstrap "smart-wardrobe-be/services/commerce/internal/bootstrap"
	brandmodule "smart-wardrobe-be/services/commerce/internal/modules/brand"
	communitymodule "smart-wardrobe-be/services/commerce/internal/modules/community"
	dashboardmodule "smart-wardrobe-be/services/commerce/internal/modules/dashboard"
	fashionmodule "smart-wardrobe-be/services/commerce/internal/modules/fashion"
	wardrobemodule "smart-wardrobe-be/services/commerce/internal/modules/wardrobe"
	sharedevent "smart-wardrobe-be/shared/application/event"
	sharedusecase "smart-wardrobe-be/shared/application/interface/usecase"
	sharedsse "smart-wardrobe-be/shared/application/sse"
	sharedUsecase "smart-wardrobe-be/shared/application/usecase"
	sharedconfig "smart-wardrobe-be/shared/config"
	"smart-wardrobe-be/shared/infrastructure/caching"
	infra_db "smart-wardrobe-be/shared/infrastructure/db"
	"smart-wardrobe-be/shared/infrastructure/media"
	"smart-wardrobe-be/shared/infrastructure/messaging"
	"smart-wardrobe-be/shared/infrastructure/search"
	sharedsecurity "smart-wardrobe-be/shared/infrastructure/security"
	sharedlogger "smart-wardrobe-be/shared/pkg/logger"
	shared_middleware "smart-wardrobe-be/shared/presentation/middleware"
	sharedWorker "smart-wardrobe-be/shared/presentation/worker"

	"github.com/google/wire"
)

// InitializeCommerceContracts composes the remote contracts Commerce Core
// consumes (T041): Identity, Subscription and Admin from Account plus AI
// orchestration and cost readers from AI, all over gRPC.
func InitializeCommerceContracts(cfg *sharedconfig.Config) (*CommerceContracts, func(), error) {
	wire.Build(NewCommerceContracts)
	return &CommerceContracts{}, nil, nil
}

// InitializeApp composes the standalone Commerce Core runtime (T034a):
// module routers and workers, remote contracts, HTTP engine and shared auth.
// Commerce hosts no gRPC service.
func InitializeApp(cfg *sharedconfig.Config, l sharedlogger.Interface) (*servicebootstrap.App, func(), error) {
	wire.Build(
		// Commerce modules (wardrobe, fashion, brand, dashboard).
		brandmodule.ProviderSet,
		communitymodule.ProviderSet,
		dashboardmodule.ProviderSet,
		fashionmodule.ProviderSet,
		wardrobemodule.ProviderSet,

		// Remote contracts (Account + AI) over gRPC.
		NewCommerceContracts,
		wire.FieldsOf(new(*CommerceContracts),
			"UserBrief",
			"UserDashboardReader",
			"UserSubscription",
			"UserQuota",
			"SubscriptionDashboard",
			"AuditLog",
			"AIOrchestrator",
			"AICostDashboardReader",
		),

		// Shared infrastructure singletons.
		infra_db.NewPostgresConnection,
		caching.NewRedisConnection,
		caching.NewRedisCacheService,
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
		wire.Bind(new(sharedevent.IMessageConsumer), new(*messaging.RabbitMQClient)),
		wire.Bind(new(sharedevent.IEventPublisher), new(*messaging.RabbitMQClient)),
		wire.Bind(new(sharedevent.IRawMessagePublisher), new(*messaging.RabbitMQClient)),
		sharedsse.NewSSEManager,
		wire.Bind(new(sharedsse.ISSEManager), new(*sharedsse.SSEManager)),
		infra_db.NewGormUnitOfWork,
		infra_db.NewPostgresJobLocker,

		// Shared auth (local verification, Redis revocation).
		sharedsecurity.NewRedisTokenBlacklistService,
		shared_middleware.NewAuthMiddleware,
		wire.Bind(new(identitycontract.IAuthMiddleware), new(*shared_middleware.AuthMiddleware)),

		// HTTP engine + runtime assembly.
		shared_middleware.NewRateLimitMiddleware,
		shared_middleware.NewBodyLimitMiddleware,
		servicebootstrap.NewEngine,
		provideMigrationsFS,
		provideValidator,
		wire.Struct(new(servicebootstrap.Workers), "*"),
		servicebootstrap.NewApp,
	)
	return &servicebootstrap.App{}, nil, nil
}
