package messaging

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	wardrobe_event "smart-wardrobe-be/internal/modules/wardrobe/application/interface/event"
	"smart-wardrobe-be/internal/pkg/logger"
	shared_event "smart-wardrobe-be/internal/shared/application/event"
	shared_msg "smart-wardrobe-be/internal/shared/infrastructure/messaging"

	"go.uber.org/zap"
)

type SearchSyncEventConsumer struct {
	messageClient shared_event.IMessageConsumer
	logger        logger.Interface
}

func NewSearchSyncEventConsumer(messageClient shared_event.IMessageConsumer, l logger.Interface) wardrobe_event.ISearchSyncEventConsumer {
	return &SearchSyncEventConsumer{messageClient: messageClient, logger: l}
}

func (c *SearchSyncEventConsumer) ConsumeEvents(ctx context.Context, handler func(ctx context.Context, event dto.WardrobeEventPayload) error) error {
	wrappedHandler := func(msg shared_event.Message) error {
		var eventPayload dto.WardrobeEventPayload
		if err := json.Unmarshal(msg.Body, &eventPayload); err != nil {
			c.logger.Error("[SearchSyncEventConsumer] Failed to unmarshal message body", zap.Error(err))
			return err
		}

		if err := handler(ctx, eventPayload); err != nil {
			c.logger.Error("[SearchSyncEventConsumer] Error processing sync event", zap.Error(err))
			if strings.Contains(err.Error(), "elasticsearch is offline") {
				c.logger.Info("[SearchSyncEventConsumer] Elasticsearch is offline. Sleeping 5s and requeuing event for retry.")
				time.Sleep(5 * time.Second)
				return err
			}
			return err
		}
		return nil
	}

	c.messageClient.RegisterConsumer(shared_msg.QueueElasticsearchSync, wrappedHandler)
	c.logger.Info("[SearchSyncEventConsumer] Registered consumer for queue", zap.String("queue", shared_msg.QueueElasticsearchSync))
	return nil
}
