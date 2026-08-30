package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"smart-wardrobe-be/internal/shared/application/event"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

// PublishDirect bypasses the outbox (no durability) — MUST only be called by
// OutboxUseCase. Every other caller MUST use Publish(), which writes to outbox
// and keeps the publish durable across broker outages.
func (r *RabbitMQClient) PublishDirect(ctx context.Context, topic string, body []byte, headers map[string]any) error {
	r.publisherMu.Lock()
	defer r.publisherMu.Unlock()

	r.mu.RLock()
	isReconnecting := r.isReconnecting
	ch := r.publisherCh
	r.mu.RUnlock()

	if isReconnecting || ch == nil {
		return fmt.Errorf("RabbitMQ client is offline or reconnecting")
	}

	r.logger.Info("Publishing event directly to RabbitMQ topic exchange",
		zap.String("routing_key", topic),
		zap.String("exchange", ExchangeName),
	)

	correlationID := ""
	if headers != nil {
		if v, ok := headers["x-correlation-id"]; ok {
			if s, ok := v.(string); ok {
				correlationID = s
			}
		} else if v, ok := headers["correlationId"]; ok {
			if s, ok := v.(string); ok {
				correlationID = s
			}
		}
	}
	err := ch.PublishWithContext(ctx,
		ExchangeName,
		topic,
		false,
		false,
		amqp.Publishing{
			DeliveryMode:  amqp.Persistent,
			ContentType:   "application/json",
			Body:          body,
			Headers:       amqp.Table(headers),
			CorrelationId: correlationID,
			MessageId:     uuid.NewString(),
			Timestamp:     time.Now().UTC(),
		},
	)
	if err != nil {
		incPublishFailed()
		return err
	}
	incPublishTotal()

	// Publish chỉ thành công khi broker xác nhận (publisher confirm);
	// Nếu nack hoặc timeout, trả lỗi để OutboxUseCase MarkFailed và retry sau.
	if err := r.waitForPublishConfirm(ctx); err != nil {
		incPublishFailed()
		return fmt.Errorf("publisher confirm failed for topic %s: %w", topic, err)
	}

	return nil
}

// publishConfirmTimeout là thời gian tối đa chờ broker xác nhận 1 publish.
const publishConfirmTimeout = 5 * time.Second

func (r *RabbitMQClient) waitForPublishConfirm(ctx context.Context) error {
	r.mu.RLock()
	confirms := r.confirms
	r.mu.RUnlock()

	if confirms == nil {
		// Confirm mode không bật được (đã warn khi connect) — không có gì để chờ.
		return nil
	}

	select {
	case c, ok := <-confirms:
		if !ok {
			return fmt.Errorf("confirm channel closed (connection may be down)")
		}
		if !c.Ack {
			return fmt.Errorf("broker nacked publish (deliveryTag %d)", c.DeliveryTag)
		}
		return nil
	case <-ctx.Done():
		return fmt.Errorf("cancelled while waiting for publisher confirm: %w", ctx.Err())
	case <-time.After(publishConfirmTimeout):
		return fmt.Errorf("timeout after %s waiting for publisher confirm", publishConfirmTimeout)
	}
}

func (r *RabbitMQClient) Publish(ctx context.Context, topic string, payload any) error {
	if r.outboxRepo == nil {
		return fmt.Errorf("outbox repository is not configured")
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	correlationID := uuid.NewString()
	headersJSON, _ := json.Marshal(map[string]any{
		"correlationId": correlationID,
		"eventVersion":  "v1",
		"publishedAt":   time.Now().UTC().Format(time.RFC3339Nano),
		"producer":      "unknown",
	})

	record := &OutboxRecord{
		Topic:   topic,
		Payload: body,
		Headers: headersJSON,
		Status:  OutboxStatusPending,
	}

	r.logger.Info("Writing outbox record for event",
		zap.String("topic", topic),
	)

	if err := r.outboxRepo.Create(ctx, record); err != nil {
		return fmt.Errorf("failed to create outbox record: %w", err)
	}

	return nil
}

// Consume returns a technology-agnostic channel of event.Message for the queue,
// wrapping the underlying broker deliveries so callers never depend on AMQP types.
func (r *RabbitMQClient) Consume(queueName string) (<-chan event.Message, error) {
	deliveries, err := r.consumeAMQP(queueName)
	if err != nil {
		return nil, err
	}
	out := make(chan event.Message)
	go func() {
		defer close(out)
		for d := range deliveries {
			out <- event.Message{Body: d.Body}
		}
	}()
	return out, nil
}

// consumeAMQP subscribes to the queue and returns raw AMQP deliveries.
// Each consumer gets its own channel with per-consumer QoS (prefetch).
func (r *RabbitMQClient) consumeAMQP(queueName string) (<-chan amqp.Delivery, error) {
	r.mu.RLock()
	conn := r.conn
	r.mu.RUnlock()

	if conn == nil {
		return nil, fmt.Errorf("RabbitMQ connection is not open")
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open channel for queue %s: %w", queueName, err)
	}

	prefetch := 20
	if r.cfg != nil && r.cfg.RabbitMQ.Prefetch > 0 {
		prefetch = r.cfg.RabbitMQ.Prefetch
	}
	if err := ch.Qos(prefetch, 0, false); err != nil {
		r.logger.Warn("Failed to set QoS on RabbitMQ channel", zap.String("queue", queueName), zap.Error(err))
	}

	return ch.Consume(
		queueName,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
}
