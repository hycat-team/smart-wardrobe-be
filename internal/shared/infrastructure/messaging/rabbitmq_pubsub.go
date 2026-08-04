package messaging

import (
	"context"
	"encoding/json"
	"fmt"

	"smart-wardrobe-be/internal/shared/application/event"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

func (r *RabbitMQClient) PublishDirect(ctx context.Context, topic string, body []byte) error {
	r.mu.Lock()

	if r.isReconnecting || r.ch == nil {
		r.mu.Unlock()
		return fmt.Errorf("RabbitMQ client is offline or reconnecting")
	}

	r.logger.Info("Publishing event directly to RabbitMQ topic exchange",
		zap.String("routing_key", topic),
		zap.String("exchange", ExchangeName),
	)

	err := r.ch.PublishWithContext(ctx,
		ExchangeName,
		topic,
		false,
		false,
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/json",
			Body:         body,
		},
	)
	r.mu.Unlock()

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

	record := &OutboxRecord{
		Topic:   topic,
		Payload: body,
		Headers: json.RawMessage(`{}`),
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
// Kept private: ack/nack handling stays inside the infrastructure layer
// (startConsumerLoop), so workers never touch broker-specific types.
func (r *RabbitMQClient) consumeAMQP(queueName string) (<-chan amqp.Delivery, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.ch == nil {
		return nil, fmt.Errorf("RabbitMQ channel is not open")
	}

	err := r.ch.Qos(
		1,     // prefetch count
		0,     // prefetch size
		false, // global
	)
	if err != nil {
		r.logger.Warn("Failed to set QoS on RabbitMQ channel", zap.Error(err))
	}

	return r.ch.Consume(
		queueName,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
}
