package messaging

import (
	"context"
	"encoding/json"
	"strings"

	fashionDto "smart-wardrobe-be/internal/modules/fashion/application/dto"
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
			return shared_msg.PermanentError{Err: err}
		}

		if err := handler(ctx, eventPayload); err != nil {
			c.logger.Error("[SearchSyncEventConsumer] Error processing sync event", zap.Error(err))
			if strings.Contains(err.Error(), "elasticsearch is offline") {
				c.logger.Info("[SearchSyncEventConsumer] Elasticsearch is offline. Requeuing for retry.")
				return shared_msg.RetryableError{Err: err}
			}
			return shared_msg.RetryableError{Err: err}
		}
		return nil
	}

	c.messageClient.RegisterConsumer(shared_msg.QueueElasticsearchSync, wrappedHandler)
	c.logger.Info("[SearchSyncEventConsumer] Registered consumer for queue", zap.String("queue", shared_msg.QueueElasticsearchSync))
	return nil
}

type WardrobeFashionAnalyzedEventConsumer struct {
	messageClient shared_event.IMessageConsumer
	logger        logger.Interface
}

func NewWardrobeFashionAnalyzedEventConsumer(messageClient shared_event.IMessageConsumer, l logger.Interface) wardrobe_event.IWardrobeFashionAnalyzedEventConsumer {
	return &WardrobeFashionAnalyzedEventConsumer{messageClient: messageClient, logger: l}
}

func (c *WardrobeFashionAnalyzedEventConsumer) ConsumeEvents(ctx context.Context, handler func(ctx context.Context, event fashionDto.FashionAnalyzedEventPayload) error) error {
	wrappedHandler := func(msg shared_event.Message) error {
		var eventPayload fashionDto.FashionAnalyzedEventPayload
		if err := json.Unmarshal(msg.Body, &eventPayload); err != nil {
			c.logger.Error("[WardrobeFashionAnalyzedEventConsumer] Failed to unmarshal message body", zap.Error(err))
			return shared_msg.PermanentError{Err: err}
		}

		if err := handler(ctx, eventPayload); err != nil {
			c.logger.Error("[WardrobeFashionAnalyzedEventConsumer] Error processing fashion analyzed event", zap.Error(err))
			return shared_msg.RetryableError{Err: err}
		}
		return nil
	}

	c.messageClient.RegisterConsumer(shared_msg.QueueWardrobeFashionAnalyzed, wrappedHandler)
	c.logger.Info("[WardrobeFashionAnalyzedEventConsumer] Registered consumer for queue", zap.String("queue", shared_msg.QueueWardrobeFashionAnalyzed))
	return nil
}

type WardrobeCleanupEventConsumer struct {
	messageClient shared_event.IMessageConsumer
	logger        logger.Interface
}

func NewWardrobeCleanupEventConsumer(messageClient shared_event.IMessageConsumer, l logger.Interface) wardrobe_event.IWardrobeCleanupEventConsumer {
	return &WardrobeCleanupEventConsumer{messageClient: messageClient, logger: l}
}

func (c *WardrobeCleanupEventConsumer) ConsumeEvents(ctx context.Context, handler func(ctx context.Context, event dto.WardrobeEventPayload) error) error {
	wrappedHandler := func(msg shared_event.Message) error {
		var eventPayload dto.WardrobeEventPayload
		if err := json.Unmarshal(msg.Body, &eventPayload); err != nil {
			c.logger.Error("[WardrobeCleanupEventConsumer] Failed to unmarshal message body", zap.Error(err))
			return shared_msg.PermanentError{Err: err}
		}
		if err := handler(ctx, eventPayload); err != nil {
			c.logger.Error("[WardrobeCleanupEventConsumer] Error processing cleanup event", zap.Error(err))
			return shared_msg.RetryableError{Err: err}
		}
		return nil
	}
	c.messageClient.RegisterConsumer(shared_msg.QueueWardrobeCleanup, wrappedHandler)
	c.logger.Info("[WardrobeCleanupEventConsumer] Registered consumer for queue", zap.String("queue", shared_msg.QueueWardrobeCleanup))
	return nil
}

type WardrobeProcessingExhaustedEventConsumer struct {
	messageClient shared_event.IMessageConsumer
	logger        logger.Interface
}

func NewWardrobeProcessingExhaustedEventConsumer(messageClient shared_event.IMessageConsumer, l logger.Interface) wardrobe_event.IWardrobeProcessingExhaustedEventConsumer {
	return &WardrobeProcessingExhaustedEventConsumer{messageClient: messageClient, logger: l}
}

func (c *WardrobeProcessingExhaustedEventConsumer) ConsumeEvents(ctx context.Context, handler func(ctx context.Context, event fashionDto.FashionProcessingExhaustedPayload) error) error {
	wrappedHandler := func(msg shared_event.Message) error {
		var eventPayload fashionDto.FashionProcessingExhaustedPayload
		if err := json.Unmarshal(msg.Body, &eventPayload); err != nil {
			c.logger.Error("[WardrobeProcessingExhaustedEventConsumer] Failed to unmarshal message body", zap.Error(err))
			return shared_msg.PermanentError{Err: err}
		}
		if err := handler(ctx, eventPayload); err != nil {
			c.logger.Error("[WardrobeProcessingExhaustedEventConsumer] Error processing processing-exhausted event", zap.Error(err))
			return shared_msg.RetryableError{Err: err}
		}
		return nil
	}
	c.messageClient.RegisterConsumer(shared_msg.QueueWardrobeProcessingExhausted, wrappedHandler)
	c.logger.Info("[WardrobeProcessingExhaustedEventConsumer] Registered consumer for queue", zap.String("queue", shared_msg.QueueWardrobeProcessingExhausted))
	return nil
}

type WardrobeFashionBindEventConsumer struct {
	messageClient shared_event.IMessageConsumer
	logger        logger.Interface
}

func NewWardrobeFashionBindEventConsumer(messageClient shared_event.IMessageConsumer, l logger.Interface) wardrobe_event.IWardrobeFashionBindEventConsumer {
	return &WardrobeFashionBindEventConsumer{messageClient: messageClient, logger: l}
}

func (c *WardrobeFashionBindEventConsumer) ConsumeEvents(ctx context.Context, handler func(ctx context.Context, event fashionDto.FashionItemCreatedPayload) error) error {
	wrappedHandler := func(msg shared_event.Message) error {
		var eventPayload fashionDto.FashionItemCreatedPayload
		if err := json.Unmarshal(msg.Body, &eventPayload); err != nil {
			c.logger.Error("[WardrobeFashionBindEventConsumer] Failed to unmarshal message body", zap.Error(err))
			return shared_msg.PermanentError{Err: err}
		}
		if err := handler(ctx, eventPayload); err != nil {
			c.logger.Error("[WardrobeFashionBindEventConsumer] Error processing fashion bind event", zap.Error(err))
			return shared_msg.RetryableError{Err: err}
		}
		return nil
	}
	c.messageClient.RegisterConsumer(shared_msg.QueueWardrobeFashionBind, wrappedHandler)
	c.logger.Info("[WardrobeFashionBindEventConsumer] Registered consumer for queue", zap.String("queue", shared_msg.QueueWardrobeFashionBind))
	return nil
}

type ElasticsearchProcessingExhaustedEventConsumer struct {
	messageClient shared_event.IMessageConsumer
	logger        logger.Interface
}

func NewElasticsearchProcessingExhaustedEventConsumer(messageClient shared_event.IMessageConsumer, l logger.Interface) wardrobe_event.IElasticsearchProcessingExhaustedEventConsumer {
	return &ElasticsearchProcessingExhaustedEventConsumer{messageClient: messageClient, logger: l}
}

func (c *ElasticsearchProcessingExhaustedEventConsumer) ConsumeEvents(ctx context.Context, handler func(ctx context.Context, event fashionDto.FashionProcessingExhaustedPayload) error) error {
	wrappedHandler := func(msg shared_event.Message) error {
		var eventPayload fashionDto.FashionProcessingExhaustedPayload
		if err := json.Unmarshal(msg.Body, &eventPayload); err != nil {
			c.logger.Error("[ElasticsearchProcessingExhaustedEventConsumer] Failed to unmarshal message body", zap.Error(err))
			return shared_msg.PermanentError{Err: err}
		}
		if err := handler(ctx, eventPayload); err != nil {
			c.logger.Error("[ElasticsearchProcessingExhaustedEventConsumer] Error processing processing-exhausted event for search", zap.Error(err))
			if strings.Contains(err.Error(), "elasticsearch is offline") {
				return shared_msg.RetryableError{Err: err}
			}
			return shared_msg.RetryableError{Err: err}
		}
		return nil
	}
	c.messageClient.RegisterConsumer(shared_msg.QueueElasticsearchProcessingExhausted, wrappedHandler)
	c.logger.Info("[ElasticsearchProcessingExhaustedEventConsumer] Registered consumer for queue", zap.String("queue", shared_msg.QueueElasticsearchProcessingExhausted))
	return nil
}
