//go:build wireinject
// +build wireinject

package providers

import (
	sharedevent "smart-wardrobe-be/shared/application/event"
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

// InitializeAccount composes the Account service (identity, subscription,
// admin, notification) with in-process dependencies for monolith assembly.
// Shared connections (db, redis, rabbitmq, search) are built once by the root
// injector and passed in — never rebuilt here.
func InitializeAccount(
	cfg *sharedconfig.Config,
	l sharedlogger.Interface,
	gormDB *gorm.DB,
	redisClient *redis.Client,
	rabbitMQClient *messaging.RabbitMQClient,
	esClient *sharedsearch.ElasticsearchClient,
) (*AccountBundle, func(), error) {
	wire.Build(
		ProviderSet,
		db.NewGormUnitOfWork,
		media.NewCloudinaryService,
		infra_caching.NewRedisCacheService,
		infra_caching.NewRedisCooldownService,
		messaging.NewOutboxRepository,
		wire.Bind(new(sharedevent.IMessageConsumer), new(*messaging.RabbitMQClient)),
		wire.Bind(new(sharedevent.IEventPublisher), new(*messaging.RabbitMQClient)),
		wire.Struct(new(AccountBundle), "*"),
	)
	return &AccountBundle{}, nil, nil
}
