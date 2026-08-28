package messaging

import (
	"context"
	"encoding/json"

	brand_event "smart-wardrobe-be/internal/modules/brand/application/interface/event"
	fashion_dto "smart-wardrobe-be/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/internal/pkg/logger"
	shared_event "smart-wardrobe-be/internal/shared/application/event"
	shared_msg "smart-wardrobe-be/internal/shared/infrastructure/messaging"

	"go.uber.org/zap"
)

type BrandFashionAnalyzedEventConsumer struct {
	messageClient shared_event.IMessageConsumer
	logger        logger.Interface
}

func NewBrandFashionAnalyzedEventConsumer(messageClient shared_event.IMessageConsumer, l logger.Interface) brand_event.IBrandFashionAnalyzedEventConsumer {
	return &BrandFashionAnalyzedEventConsumer{messageClient: messageClient, logger: l}
}

func (c *BrandFashionAnalyzedEventConsumer) ConsumeEvents(ctx context.Context, handler func(ctx context.Context, event fashion_dto.FashionAnalyzedEventPayload) error) error {
	wrappedHandler := func(msg shared_event.Message) error {
		var eventPayload fashion_dto.FashionAnalyzedEventPayload
		if err := json.Unmarshal(msg.Body, &eventPayload); err != nil {
			c.logger.Error("[BrandFashionAnalyzedEventConsumer] Failed to unmarshal message body", zap.Error(err))
			return shared_msg.PermanentError{Err: err}
		}

		if err := handler(ctx, eventPayload); err != nil {
			c.logger.Error("[BrandFashionAnalyzedEventConsumer] Error processing brand fashion analyzed event", zap.Error(err))
			return shared_msg.RetryableError{Err: err}
		}
		return nil
	}

	c.messageClient.RegisterConsumer(shared_msg.QueueBrandFashionAnalyzed, wrappedHandler)
	c.logger.Info("[BrandFashionAnalyzedEventConsumer] Registered consumer for queue", zap.String("queue", shared_msg.QueueBrandFashionAnalyzed))
	return nil
}

type BrandFashionBindEventConsumer struct {
	messageClient shared_event.IMessageConsumer
	logger        logger.Interface
}

func NewBrandFashionBindEventConsumer(messageClient shared_event.IMessageConsumer, l logger.Interface) brand_event.IBrandFashionBindEventConsumer {
	return &BrandFashionBindEventConsumer{messageClient: messageClient, logger: l}
}

func (c *BrandFashionBindEventConsumer) ConsumeEvents(ctx context.Context, handler func(ctx context.Context, event fashion_dto.FashionItemCreatedPayload) error) error {
	wrappedHandler := func(msg shared_event.Message) error {
		var eventPayload fashion_dto.FashionItemCreatedPayload
		if err := json.Unmarshal(msg.Body, &eventPayload); err != nil {
			c.logger.Error("[BrandFashionBindEventConsumer] Failed to unmarshal message body", zap.Error(err))
			return shared_msg.PermanentError{Err: err}
		}
		if err := handler(ctx, eventPayload); err != nil {
			c.logger.Error("[BrandFashionBindEventConsumer] Error processing brand fashion bind event", zap.Error(err))
			return shared_msg.RetryableError{Err: err}
		}
		return nil
	}
	c.messageClient.RegisterConsumer(shared_msg.QueueBrandFashionBind, wrappedHandler)
	c.logger.Info("[BrandFashionBindEventConsumer] Registered consumer for queue", zap.String("queue", shared_msg.QueueBrandFashionBind))
	return nil
}
