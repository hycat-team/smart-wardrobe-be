package messaging

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"smart-wardrobe-be/shared/application/event"
	"smart-wardrobe-be/shared/config"
	"smart-wardrobe-be/shared/infrastructure/resilience"
	"smart-wardrobe-be/shared/pkg/logger"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

var _ event.IEventPublisher = (*RabbitMQClient)(nil)
var _ event.IMessageConsumer = (*RabbitMQClient)(nil)

type consumerRegistration struct {
	queueName string
	handler   func(event.Message) error
}

type RabbitMQClient struct {
	cfg            *config.Config
	logger         logger.Interface
	conn           *amqp.Connection
	ch             *amqp.Channel
	publisherCh    *amqp.Channel
	confirms       <-chan amqp.Confirmation
	mu             sync.RWMutex
	publisherMu    sync.Mutex
	isReconnecting bool
	consumers      []consumerRegistration
	consumerMu     sync.Mutex
	outboxRepo     OutboxRepository
	baseCtx        context.Context
	baseCancel     context.CancelFunc
}

func NewRabbitMQClient(cfg *config.Config, l logger.Interface, outboxRepo OutboxRepository) (*RabbitMQClient, error) {
	client := &RabbitMQClient{
		cfg:        cfg,
		logger:     l,
		outboxRepo: outboxRepo,
	}
	client.baseCtx, client.baseCancel = context.WithCancel(context.Background())

	err := resilience.RunStartupRetry(cfg, l, "rabbitmq", func(timeout time.Duration) error {
		return client.connect(timeout)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize RabbitMQ client: %w", err)
	}
	return client, nil
}

func (r *RabbitMQClient) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.baseCancel != nil {
		r.baseCancel()
	}
	if r.publisherCh != nil {
		_ = r.publisherCh.Close()
	}
	if r.ch != nil {
		_ = r.ch.Close()
	}
	if r.conn != nil {
		_ = r.conn.Close()
	}
	r.logger.Info("Closed RabbitMQ connections successfully.")
}

func (r *RabbitMQClient) RegisterConsumer(queueName string, handler func(event.Message) error) {
	r.consumerMu.Lock()

	r.consumers = append(r.consumers, consumerRegistration{
		queueName: queueName,
		handler:   handler,
	})

	deliveries, err := r.consumeAMQP(queueName)
	r.consumerMu.Unlock()

	if err != nil {
		r.logger.Error("Failed to register consumer for queue", zap.String("queue", queueName), zap.Error(err))
		return
	}
	go r.startConsumerLoop(queueName, handler, deliveries)
	r.logger.Debug("Registered consumer for queue", zap.String("queue", queueName))
}

func (r *RabbitMQClient) reRegisterConsumers() {
	r.consumerMu.Lock()
	defer r.consumerMu.Unlock()

	for _, reg := range r.consumers {
		deliveries, err := r.consumeAMQP(reg.queueName)
		if err != nil {
			r.logger.Error("Failed to re-register consumer for queue", zap.String("queue", reg.queueName), zap.Error(err))
			continue
		}
		go r.startConsumerLoop(reg.queueName, reg.handler, deliveries)
		r.logger.Info("Re-registered consumer for queue", zap.String("queue", reg.queueName))
	}
}

func (r *RabbitMQClient) startConsumerLoop(queueName string, handler func(event.Message) error, deliveries <-chan amqp.Delivery) {
	for d := range deliveries {
		msg := event.Message{Body: d.Body, Headers: amqpTableToMap(d.Headers)}
		if err := handler(msg); err != nil {
			r.logger.Error("Consumer handler error", zap.String("queue", queueName), zap.Error(err))
			incNack()
			var pe event.PermanentError
			var re event.RetryableError
			switch {
			case errors.As(err, &pe):
				_ = d.Nack(false, false)
			case errors.As(err, &re):
				_ = d.Nack(false, true)
			default:
				_ = d.Nack(false, false)
			}
		} else {
			incAck()
			_ = d.Ack(false)
		}
	}
}

func amqpTableToMap(t amqp.Table) map[string]any {
	if t == nil {
		return nil
	}
	m := make(map[string]any, len(t))
	for k, v := range t {
		m[k] = v
	}
	return m
}
