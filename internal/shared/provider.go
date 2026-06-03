package shared

import (
	"smart-wardrobe-be/internal/shared/application/event"
	infra_ai "smart-wardrobe-be/internal/shared/infrastructure/ai"
	infra_media "smart-wardrobe-be/internal/shared/infrastructure/media"
	"smart-wardrobe-be/internal/shared/infrastructure/messaging"
	"smart-wardrobe-be/internal/shared/infrastructure/search"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	infra_media.NewCloudinaryService,
	infra_ai.NewAIService,
	messaging.NewRabbitMQClient,
	search.NewElasticsearchClient,
	wire.Bind(new(messaging.IRabbitMQClient), new(*messaging.RabbitMQClient)),
	wire.Bind(new(event.IEventPublisher), new(*messaging.RabbitMQClient)),
)
