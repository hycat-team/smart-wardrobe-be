package shared

import (
	"smart-wardrobe-be/shared/application/event"
	sharedUcInterface "smart-wardrobe-be/shared/application/interface/usecase"
	"smart-wardrobe-be/shared/application/sse"
	sharedUsecase "smart-wardrobe-be/shared/application/usecase"
	infra_ai "smart-wardrobe-be/shared/infrastructure/ai"
	infra_caching "smart-wardrobe-be/shared/infrastructure/caching"
	infra_db "smart-wardrobe-be/shared/infrastructure/db"
	infra_media "smart-wardrobe-be/shared/infrastructure/media"
	"smart-wardrobe-be/shared/infrastructure/messaging"
	"smart-wardrobe-be/shared/infrastructure/search"
	sharedWorker "smart-wardrobe-be/shared/presentation/worker"

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
	messaging.NewOutboxGaugeProbe,
	search.NewElasticsearchClient,
	sse.NewSSEManager,
	infra_db.NewOutboxListener,
	sharedUsecase.NewOutboxUseCase,
	sharedWorker.NewOutboxWorker,
	wire.Bind(new(event.IMessageConsumer), new(*messaging.RabbitMQClient)),
	wire.Bind(new(event.IEventPublisher), new(*messaging.RabbitMQClient)),
	wire.Bind(new(event.IRawMessagePublisher), new(*messaging.RabbitMQClient)),
	wire.Bind(new(event.IOutboxGaugeProbe), new(*messaging.OutboxGaugeProbe)),
	wire.Bind(new(sharedWorker.OutboxSignalListener), new(*infra_db.OutboxListener)),
	wire.Bind(new(sse.ISSEManager), new(*sse.SSEManager)),
	wire.Bind(new(sharedUcInterface.IOutboxUseCase), new(*sharedUsecase.OutboxUseCase)),
)
