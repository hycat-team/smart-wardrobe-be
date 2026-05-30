package messaging

import (
	"context"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

func (r *RabbitMQClient) Publish(ctx context.Context, topic string, payload interface{}) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.isReconnecting || r.ch == nil {
		return fmt.Errorf("RabbitMQ client is offline or reconnecting")
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal rabbitmq payload: %w", err)
	}

	r.logger.Info("Publishing event to RabbitMQ topic exchange",
		zap.String("routing_key", topic),
		zap.String("exchange", ExchangeName),
	)

	return r.ch.PublishWithContext(ctx,
		ExchangeName, // exchange
		topic,        // routing key (tên topic/queue)
		false,        // mandatory
		false,        // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent, // Tin nhắn bền vững
			ContentType:  "application/json",
			Body:         body,
		},
	)
}

func (r *RabbitMQClient) Consume(queueName string) (<-chan amqp.Delivery, error) {
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
