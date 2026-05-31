package messaging

import (
	"context"
	"encoding/json"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/application/interface/event"
	shared_msg "smart-wardrobe-be/internal/shared/infrastructure/messaging"
	"smart-wardrobe-be/pkg/logger"

	"go.uber.org/zap"
)

type BatchCropJobConsumer struct {
	messageClient shared_msg.IRabbitMQClient
	logger        logger.Interface
}

func NewBatchCropJobConsumer(messageClient shared_msg.IRabbitMQClient, l logger.Interface) event.IBatchCropJobConsumer {
	return &BatchCropJobConsumer{messageClient: messageClient, logger: l}
}

func (c *BatchCropJobConsumer) ConsumeJobs(ctx context.Context, handler func(ctx context.Context, job dto.BatchCropJobDTO) error) error {
	deliveries, err := c.messageClient.Consume(shared_msg.QueueBatchCropJobs)
	if err != nil {
		return err
	}

	go func() {
		for d := range deliveries {
			var job dto.BatchCropJobDTO
			if err := json.Unmarshal(d.Body, &job); err != nil {
				c.logger.Error("[BatchCropJobConsumer] Failed to unmarshal message body", zap.Error(err))
				_ = d.Nack(false, false)
				continue
			}

			if err := handler(ctx, job); err != nil {
				c.logger.Error("[BatchCropJobConsumer] Error processing job", zap.Error(err))
				_ = d.Nack(false, false)
			} else {
				_ = d.Ack(false)
			}
		}
	}()

	return nil
}

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
				_ = d.Nack(false, false)
			} else {
				_ = d.Ack(false)
			}
		}
	}()

	return nil
}
