package shared

import (
	infra_ai "smart-wardrobe-be/internal/shared/infrastructure/ai"
	infra_media "smart-wardrobe-be/internal/shared/infrastructure/media"
	"smart-wardrobe-be/internal/shared/infrastructure/rabbitmq"
	"smart-wardrobe-be/internal/shared/application/event"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	infra_media.NewCloudinaryService,
	infra_ai.NewAIService,
	rabbitmq.NewRabbitMQClient,
	wire.Bind(new(rabbitmq.IRabbitMQClient), new(*rabbitmq.RabbitMQClient)),
	wire.Bind(new(event.IEventPublisher), new(*rabbitmq.RabbitMQClient)),
)
