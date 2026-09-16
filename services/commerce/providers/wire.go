//go:build wireinject
// +build wireinject

package providers

import (
	accountproviders "smart-wardrobe-be/services/account/providers"
	aiproviders "smart-wardrobe-be/services/ai/providers"
	sharedevent "smart-wardrobe-be/shared/application/event"
	"smart-wardrobe-be/shared/application/sse"
	sharedsse "smart-wardrobe-be/shared/application/sse"
	sharedconfig "smart-wardrobe-be/shared/config"
	infra_caching "smart-wardrobe-be/shared/infrastructure/caching"
	"smart-wardrobe-be/shared/infrastructure/db"
	"smart-wardrobe-be/shared/infrastructure/media"
	"smart-wardrobe-be/shared/infrastructure/messaging"
	sharedsearch "smart-wardrobe-be/shared/infrastructure/search"
	sharedlogger "smart-wardrobe-be/shared/pkg/logger"

	"github.com/google/wire"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// InitializeCommerce composes the Commerce Core service (wardrobe, fashion,
// brand, dashboard) with in-process dependencies for monolith assembly.
// Account/AI-owned contracts arrive prebuilt in their bundles (DAG:
// Commerce → Account, Commerce → AI).
func InitializeCommerce(
	cfg *sharedconfig.Config,
	l sharedlogger.Interface,
	gormDB *gorm.DB,
	redisClient *redis.Client,
	rabbitMQClient *messaging.RabbitMQClient,
	esClient *sharedsearch.ElasticsearchClient,
	account *accountproviders.AccountBundle,
	ai *aiproviders.AIBundle,
) (*CommerceBundle, func(), error) {
	wire.Build(
		ProviderSet,
		db.NewGormUnitOfWork,
		media.NewCloudinaryService,
		infra_caching.NewRedisCacheService,
		infra_caching.NewRedisRateLimiter,
		sse.NewSSEManager,
		wire.Bind(new(sharedsse.ISSEManager), new(*sse.SSEManager)),
		wire.FieldsOf(new(*accountproviders.AccountBundle),
			"UserBriefContract",
			"UserDashboardReader",
			"UserSubscriptionContract",
			"UserQuotaContract",
			"SubscriptionDashboardReader",
			"AuditLogContract",
			"AuthMiddleware",
		),
		wire.FieldsOf(new(*aiproviders.AIBundle),
			"AIOrchestrator",
			"AICostDashboardReader",
		),
		wire.Bind(new(sharedevent.IMessageConsumer), new(*messaging.RabbitMQClient)),
		wire.Bind(new(sharedevent.IEventPublisher), new(*messaging.RabbitMQClient)),
		wire.Struct(new(CommerceBundle), "*"),
	)
	return &CommerceBundle{}, nil, nil
}
