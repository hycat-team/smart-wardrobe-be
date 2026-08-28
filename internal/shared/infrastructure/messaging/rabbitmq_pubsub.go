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

// PublishDirect publishes directly to RabbitMQ — MUST only be called by OutboxUseCase (constitution §III, FR-018).
// Other usecases must use Publish() which writes to outbox.
func (r *RabbitMQClient) PublishDirect(ctx context.Context, topic string, body []byte, headers amqp.Table) error {
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
			Headers:       headers,
			CorrelationId: correlationID,
			MessageId:     uuid.NewString(),
			Timestamp:     time.Now().UTC(),
		},
	)
	if err != nil {
		incPublishFailed()
	} else {
		incPublishTotal()
	}

	return err
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
