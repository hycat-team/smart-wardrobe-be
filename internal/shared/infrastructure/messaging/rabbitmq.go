package messaging

import (
	"fmt"
	"sync"
	"time"

	"smart-wardrobe-be/internal/config"
	"smart-wardrobe-be/internal/pkg/logger"
	"smart-wardrobe-be/internal/shared/application/event"
	"smart-wardrobe-be/internal/shared/infrastructure/resilience"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

var _ event.IEventPublisher = (*RabbitMQClient)(nil)

type IRabbitMQClient interface {
	Consume(queueName string) (<-chan amqp.Delivery, error)
	RegisterConsumer(queueName string, handler func(delivery amqp.Delivery) error)
	Close()
}

type consumerRegistration struct {
	queueName string
	handler   func(delivery amqp.Delivery) error
}

type RabbitMQClient struct {
	cfg            *config.Config
	logger         logger.Interface
	conn           *amqp.Connection
	ch             *amqp.Channel
	mu             sync.RWMutex
	isReconnecting bool
	consumers      []consumerRegistration
	consumerMu     sync.Mutex
	outboxRepo     OutboxRepository
}

func NewRabbitMQClient(cfg *config.Config, l logger.Interface, outboxRepo OutboxRepository) (*RabbitMQClient, error) {
	client := &RabbitMQClient{
		cfg:        cfg,
		logger:     l,
		outboxRepo: outboxRepo,
	}

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

	if r.ch != nil {
		_ = r.ch.Close()
	}
	if r.conn != nil {
		_ = r.conn.Close()
	}
	r.logger.Info("Closed RabbitMQ connections successfully.")
}

func (r *RabbitMQClient) RegisterConsumer(queueName string, handler func(delivery amqp.Delivery) error) {
	r.consumerMu.Lock()

	r.consumers = append(r.consumers, consumerRegistration{
		queueName: queueName,
		handler:   handler,
	})

	deliveries, err := r.Consume(queueName)
	r.consumerMu.Unlock()

	if err != nil {
		r.logger.Error("Failed to register consumer for queue", zap.String("queue", queueName), zap.Error(err))
		return
	}
	go r.startConsumerLoop(queueName, handler, deliveries)
	r.logger.Info("Registered consumer for queue", zap.String("queue", queueName))
}

func (r *RabbitMQClient) reRegisterConsumers() {
	r.consumerMu.Lock()
	defer r.consumerMu.Unlock()

	for _, reg := range r.consumers {
		deliveries, err := r.Consume(reg.queueName)
		if err != nil {
			r.logger.Error("Failed to re-register consumer for queue", zap.String("queue", reg.queueName), zap.Error(err))
			continue
		}
		go r.startConsumerLoop(reg.queueName, reg.handler, deliveries)
		r.logger.Info("Re-registered consumer for queue", zap.String("queue", reg.queueName))
	}
}

func (r *RabbitMQClient) startConsumerLoop(queueName string, handler func(amqp.Delivery) error, deliveries <-chan amqp.Delivery) {
	for d := range deliveries {
		if err := handler(d); err != nil {
			r.logger.Error("Consumer handler error", zap.String("queue", queueName), zap.Error(err))
			_ = d.Nack(false, false)
		} else {
			_ = d.Ack(false)
		}
	}
}
