package messaging

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/application/interface/event"
	shared_msg "smart-wardrobe-be/internal/shared/infrastructure/messaging"
	"smart-wardrobe-be/pkg/logger"

	"go.uber.org/zap"
)

type SearchSyncEventConsumer struct {
	messageClient shared_msg.IRabbitMQClient
	logger        logger.Interface
}

func NewSearchSyncEventConsumer(messageClient shared_msg.IRabbitMQClient, l logger.Interface) event.ISearchSyncEventConsumer {
	return &SearchSyncEventConsumer{messageClient: messageClient, logger: l}
}

func (c *SearchSyncEventConsumer) ConsumeEvents(ctx context.Context, handler func(ctx context.Context, event dto.WardrobeEventPayload) error) error {
	deliveries, err := c.messageClient.Consume(shared_msg.QueueElasticsearchSync)
	if err != nil {
		return err
	}

	go func() {
		for d := range deliveries {
			var eventPayload dto.WardrobeEventPayload
			if err := json.Unmarshal(d.Body, &eventPayload); err != nil {
				c.logger.Error("[SearchSyncEventConsumer] Failed to unmarshal message body", zap.Error(err))
				_ = d.Nack(false, false)
				continue
			}

			if err := handler(ctx, eventPayload); err != nil {
				c.logger.Error("[SearchSyncEventConsumer] Error processing sync event", zap.Error(err))
				if strings.Contains(err.Error(), "elasticsearch is offline") {
					c.logger.Info("[SearchSyncEventConsumer] Elasticsearch is offline. Sleeping 5s and requeuing event for retry.")
					time.Sleep(5 * time.Second)
					_ = d.Nack(false, true) // Requeue
				} else {
					_ = d.Nack(false, false) // Discard poison pill
				}
			} else {
				_ = d.Ack(false)
			}
		}
	}()

	return nil
}
