package shared

import (
	"smart-wardrobe-be/internal/shared/application/event"
	sharedUcInterface "smart-wardrobe-be/internal/shared/application/interface/usecase"
	"smart-wardrobe-be/internal/shared/application/sse"
	sharedUsecase "smart-wardrobe-be/internal/shared/application/usecase"
	infra_ai "smart-wardrobe-be/internal/shared/infrastructure/ai"
	infra_caching "smart-wardrobe-be/internal/shared/infrastructure/caching"
	infra_db "smart-wardrobe-be/internal/shared/infrastructure/db"
	infra_media "smart-wardrobe-be/internal/shared/infrastructure/media"
	"smart-wardrobe-be/internal/shared/infrastructure/messaging"
	"smart-wardrobe-be/internal/shared/infrastructure/search"
	sharedWorker "smart-wardrobe-be/internal/shared/presentation/worker"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	infra_media.NewCloudinaryService,
	infra_ai.NewLocalTokenEstimatorFromConfig,
	infra_ai.NewAIService,
	infra_caching.NewRedisCacheService,
	infra_caching.NewRedisRateLimiter,
	infra_caching.NewRedisCooldownService,
	messaging.NewOutboxRepository,
	messaging.NewRabbitMQClient,
	search.NewElasticsearchClient,
	sse.NewSSEManager,
	infra_db.NewOutboxListener,
	sharedUsecase.NewOutboxUseCase,
	sharedWorker.NewOutboxWorker,
	wire.Bind(new(event.IMessageConsumer), new(*messaging.RabbitMQClient)),
	wire.Bind(new(event.IEventPublisher), new(*messaging.RabbitMQClient)),
	wire.Bind(new(sse.ISSEManager), new(*sse.SSEManager)),
	wire.Bind(new(sharedUcInterface.IOutboxUseCase), new(*sharedUsecase.OutboxUseCase)),
)
