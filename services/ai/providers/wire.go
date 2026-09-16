//go:build wireinject
// +build wireinject

package providers

import (
	accountproviders "smart-wardrobe-be/services/account/providers"
	sharedevent "smart-wardrobe-be/shared/application/event"
	sharedconfig "smart-wardrobe-be/shared/config"
	infra_ai "smart-wardrobe-be/shared/infrastructure/ai"
	infra_caching "smart-wardrobe-be/shared/infrastructure/caching"
	"smart-wardrobe-be/shared/infrastructure/db"
	"smart-wardrobe-be/shared/infrastructure/messaging"
	sharedsearch "smart-wardrobe-be/shared/infrastructure/search"
	sharedlogger "smart-wardrobe-be/shared/pkg/logger"

	"github.com/google/wire"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// InitializeAI composes the AI Engine service (aiexecutor) with in-process
// dependencies for monolith assembly. Account-owned contracts arrive prebuilt
// in the account bundle (DAG: AI → Account).
func InitializeAI(
	cfg *sharedconfig.Config,
	l sharedlogger.Interface,
	gormDB *gorm.DB,
	redisClient *redis.Client,
	rabbitMQClient *messaging.RabbitMQClient,
	esClient *sharedsearch.ElasticsearchClient,
	account *accountproviders.AccountBundle,
) (*AIBundle, func(), error) {
	wire.Build(
		ProviderSet,
		db.NewGormUnitOfWork,
		infra_ai.NewLocalTokenEstimatorFromConfig,
		infra_ai.NewAIService,
		infra_caching.NewRedisCacheService,
		wire.FieldsOf(new(*accountproviders.AccountBundle),
			"UserSubscriptionContract",
			"AuditLogContract",
			"AuthMiddleware",
		),
		wire.Bind(new(sharedevent.IMessageConsumer), new(*messaging.RabbitMQClient)),
		wire.Struct(new(AIBundle), "*"),
	)
	return &AIBundle{}, nil, nil
}
