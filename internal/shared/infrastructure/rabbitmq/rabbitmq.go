package rabbitmq

import (
	"fmt"
	"sync"

	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/shared/application/event"
	"smart-wardrobe-be/pkg/logger"

	amqp "github.com/rabbitmq/amqp091-go"
)

var _ event.IEventPublisher = (*RabbitMQClient)(nil)

const (
	ExchangeName = "smart_wardrobe_exchange"
	ExchangeType = "topic"
)

type IRabbitMQClient interface {
	Consume(queueName string) (<-chan amqp.Delivery, error)
	Close()
}

type RabbitMQClient struct {
	cfg            *config.Config
	logger         logger.Interface
	conn           *amqp.Connection
	ch             *amqp.Channel
	mu             sync.RWMutex
	isReconnecting bool
}

func NewRabbitMQClient(cfg *config.Config, l logger.Interface) (*RabbitMQClient, error) {
	client := &RabbitMQClient{
		cfg:    cfg,
		logger: l,
	}
	err := client.connect()
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
