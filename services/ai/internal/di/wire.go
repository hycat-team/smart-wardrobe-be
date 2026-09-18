//go:build wireinject
// +build wireinject

package di

import (
	identitycontract "smart-wardrobe-be/services/account/contract/identity"
	servicebootstrap "smart-wardrobe-be/services/ai/internal/bootstrap"
	aiexecutormodule "smart-wardrobe-be/services/ai/internal/modules/aiexecutor"
	aigrpc "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/presentation/grpc"
	sharedevent "smart-wardrobe-be/shared/application/event"
	sharedusecase "smart-wardrobe-be/shared/application/interface/usecase"
	sharedUsecase "smart-wardrobe-be/shared/application/usecase"
	sharedconfig "smart-wardrobe-be/shared/config"
	infra_ai "smart-wardrobe-be/shared/infrastructure/ai"
	"smart-wardrobe-be/shared/infrastructure/caching"
	infra_db "smart-wardrobe-be/shared/infrastructure/db"
	"smart-wardrobe-be/shared/infrastructure/messaging"
	"smart-wardrobe-be/shared/infrastructure/search"
	sharedsecurity "smart-wardrobe-be/shared/infrastructure/security"
	sharedlogger "smart-wardrobe-be/shared/pkg/logger"
	shared_middleware "smart-wardrobe-be/shared/presentation/middleware"
	sharedWorker "smart-wardrobe-be/shared/presentation/worker"

	"github.com/google/wire"
)

// InitializeAIContracts composes the remote contracts AI Engine consumes
// (T042): Identity, Subscription and Admin from Account, all over gRPC.
func InitializeAIContracts(cfg *sharedconfig.Config) (*AIContracts, func(), error) {
	wire.Build(NewAIContracts)
	return &AIContracts{}, nil, nil
}

// InitializeApp composes the standalone AI Engine runtime (T034a): the
// aiexecutor module, remote Account contracts, HTTP engine, gRPC servers and
// workers, selected at runtime by APP_MODE.
func InitializeApp(cfg *sharedconfig.Config, l sharedlogger.Interface) (*servicebootstrap.App, func(), error) {
	wire.Build(
		// AI module (aiexecutor).
		aiexecutormodule.ProviderSet,

		// Remote Account contracts over gRPC.
		NewAIContracts,
		wire.FieldsOf(new(*AIContracts),
			"UserSubscription",
			"AuditLog",
		),

		// Shared infrastructure singletons.
		infra_db.NewPostgresConnection,
		caching.NewRedisConnection,
		caching.NewRedisCacheService,
		caching.NewRedisRateLimiter,
		infra_ai.NewLocalTokenEstimatorFromConfig,
		infra_ai.NewAIService,
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
		wire.Bind(new(sharedevent.IRawMessagePublisher), new(*messaging.RabbitMQClient)),
		infra_db.NewGormUnitOfWork,

		// Shared auth (local verification, Redis revocation).
		sharedsecurity.NewRedisTokenBlacklistService,
		shared_middleware.NewAuthMiddleware,
		wire.Bind(new(identitycontract.IAuthMiddleware), new(*shared_middleware.AuthMiddleware)),

		// HTTP engine + runtime assembly.
		shared_middleware.NewRateLimitMiddleware,
		shared_middleware.NewBodyLimitMiddleware,
		servicebootstrap.NewEngine,
		aigrpc.NewAIOrchestratorServer,
		aigrpc.NewAIDashboardServer,
		servicebootstrap.NewGRPCServer,
		provideMigrationsFS,
		provideValidator,
		wire.Struct(new(servicebootstrap.Workers), "*"),
		servicebootstrap.NewApp,
	)
	return &servicebootstrap.App{}, nil, nil
}
