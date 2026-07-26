package shared

import (
	"smart-wardrobe-be/internal/shared/application/event"
	"smart-wardrobe-be/internal/shared/application/sse"
	infra_ai "smart-wardrobe-be/internal/shared/infrastructure/ai"
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
	messaging.NewOutboxRepository,
	messaging.NewRabbitMQClient,
	search.NewElasticsearchClient,
	sse.NewSSEManager,
	infra_db.NewOutboxListener,
	sharedWorker.NewOutboxWorker,
	wire.Bind(new(messaging.IRabbitMQClient), new(*messaging.RabbitMQClient)),
	wire.Bind(new(event.IEventPublisher), new(*messaging.RabbitMQClient)),
	wire.Bind(new(sse.ISSEManager), new(*sse.SSEManager)),
)
