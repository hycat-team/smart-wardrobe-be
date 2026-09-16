//go:build wireinject
// +build wireinject

package di

import (
	"smart-wardrobe-be/internal/api/routes"
	"smart-wardrobe-be/internal/bootstrap"
	accountproviders "smart-wardrobe-be/services/account/providers"
	aiproviders "smart-wardrobe-be/services/ai/providers"
	commerceproviders "smart-wardrobe-be/services/commerce/providers"
	sharedevent "smart-wardrobe-be/shared/application/event"
	sharedusecase "smart-wardrobe-be/shared/application/interface/usecase"
	sharedUsecase "smart-wardrobe-be/shared/application/usecase"
	sharedconfig "smart-wardrobe-be/shared/config"
	"smart-wardrobe-be/shared/infrastructure/caching"
	infra_db "smart-wardrobe-be/shared/infrastructure/db"
	"smart-wardrobe-be/shared/infrastructure/messaging"
	"smart-wardrobe-be/shared/infrastructure/search"
	"smart-wardrobe-be/shared/pkg/logger"
	shared_middleware "smart-wardrobe-be/shared/presentation/middleware"
	sharedWorker "smart-wardrobe-be/shared/presentation/worker"

	"github.com/google/wire"
)

// InitializeApp composes the transitional monolith from the shared layer and
// the three service compositions (020). Shared connections are built once here
// and passed down; each service bundle is built in-process (monolith mode).
// Removed with cmd/server at T034.
func InitializeApp(cfg *sharedconfig.Config, l logger.Interface) (*bootstrap.App, func(), error) {
	wire.Build(
		bootstrap.NewApp,
		// Shared layer singletons (built once, shared by all bundles).
		infra_db.NewPostgresConnection,
		caching.NewRedisConnection,
		messaging.NewOutboxRepository,
		messaging.NewRabbitMQClient,
		search.NewElasticsearchClient,
		messaging.NewOutboxGaugeProbe,
		sharedUsecase.NewOutboxUseCase,
		infra_db.NewOutboxListener,
		sharedWorker.NewOutboxWorker,
		wire.Bind(new(sharedusecase.IOutboxUseCase), new(*sharedUsecase.OutboxUseCase)),
		wire.Bind(new(sharedWorker.OutboxSignalListener), new(*infra_db.OutboxListener)),
		wire.Bind(new(sharedevent.IOutboxGaugeProbe), new(*messaging.OutboxGaugeProbe)),
		wire.Bind(new(sharedevent.IRawMessagePublisher), new(*messaging.RabbitMQClient)),
		// Service compositions (monolith mode, DAG: commerce/ai ← account).
		accountproviders.InitializeAccount,
		aiproviders.InitializeAI,
		commerceproviders.InitializeCommerce,
		// Bundle fields consumed by the engine and workers assembly.
		wire.FieldsOf(new(*accountproviders.AccountBundle),
			"IdentityRouter",
			"SubscriptionRouter",
			"AdminRouter",
			"SubscriptionRenewalWorker",
			"PaymentReconciliationWorker",
			"WebhookInboxWorker",
			"EmailWorker",
			"AuditLogConsumerWorker",
			"AuditCleanupWorker",
			"Validator",
		),
		wire.FieldsOf(new(*aiproviders.AIBundle),
			"AIExecutorRouter",
			"AIUsageReconciliationWorker",
			"PlanChangedWorker",
		),
		wire.FieldsOf(new(*commerceproviders.CommerceBundle),
			"WardrobeRouter",
			"DashboardRouter",
			"BrandRouter",
			"FashionRouter",
			"LoyaltyPointExpiryWorker",
			"SearchSyncWorker",
			"FashionInitWorker",
			"FailedItemsCleanupWorker",
			"ProcessingRecoveryWorker",
			"FashionAnalyzeWorker",
			"DashboardAggregatorWorker",
			"WardrobeFashionEventWorker",
			"BrandFashionEventWorker",
			"WardrobeCleanupWorker",
			"WardrobeProcessingExhaustedWorker",
			"WardrobeFashionBindWorker",
			"ElasticsearchProcessingExhaustedWorker",
			"BrandFashionBindWorker",
		),

		shared_middleware.NewRateLimitMiddleware,
		shared_middleware.NewBodyLimitMiddleware,
		routes.RouterSet,
		routes.NewEngine,
		wire.Struct(new(bootstrap.AppWorkers), "*"),
	)
	return &bootstrap.App{}, nil, nil
}
