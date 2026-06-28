package messaging

import (
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

func (r *RabbitMQClient) connect(timeout time.Duration) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	url := fmt.Sprintf("amqp://%s:%s@%s:%d/",
		r.cfg.RabbitMQ.User,
		r.cfg.RabbitMQ.Password,
		r.cfg.RabbitMQ.Host,
		r.cfg.RabbitMQ.Port,
	)

	conn, err := amqp.DialConfig(url, amqp.Config{
		Locale: "en_US",
		Dial:   amqp.DefaultDial(timeout),
	})
	if err != nil {
		return fmt.Errorf("could not connect to RabbitMQ broker: %w", err)
	}
	r.conn = conn

	r.ch, err = r.conn.Channel()
	if err != nil {
		_ = r.conn.Close()
		return fmt.Errorf("could not open RabbitMQ channel: %w", err)
	}

	// 1. Initialize flexible Topic Exchange, standard event-driven Pub/Sub
	err = r.ch.ExchangeDeclare(
		ExchangeName, // exchange name
		ExchangeType, // exchange type ("topic")
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	)
	if err != nil {
		_ = r.ch.Close()
		_ = r.conn.Close()
		return fmt.Errorf("could not declare RabbitMQ exchange: %w", err)
	}

	// 2. Declare and Bind Wardrobe Batch Upload Queue
	if err := r.DeclareAndBindQueue(QueueWardrobeBatchUpload, RoutingKeyWardrobeBatchUpload); err != nil {
		_ = r.ch.Close()
		_ = r.conn.Close()
		return err
	}

	// 3. Declare and Bind Elasticsearch Sync Queue
	if err := r.DeclareAndBindQueue(QueueElasticsearchSync, RoutingKeyElasticsearchSyncWildcard); err != nil {
		_ = r.ch.Close()
		_ = r.conn.Close()
		return err
	}

	// 4. Declare and Bind Fashion Analyze Item Queue
	if err := r.DeclareAndBindQueue(QueueFashionAnalyzeItem, RoutingKeyFashionAnalyzeItem); err != nil {
		_ = r.ch.Close()
		_ = r.conn.Close()
		return err
	}

	r.logger.Info("Successfully connected to RabbitMQ and established binding topology.")

	// 4. Register to listen to NotifyClose for auto-reconnection in the background upon disconnection
	errChan := make(chan *amqp.Error, 1)
	r.conn.NotifyClose(errChan)
	go r.handleReconnect(errChan)

	return nil
}

func (r *RabbitMQClient) handleReconnect(errChan chan *amqp.Error) {
	reason := <-errChan
	if reason != nil {
		r.logger.Warn("RabbitMQ connection disrupted. Initiating auto-reconnection...", zap.Error(reason))
		r.reconnect()
	}
}

func (r *RabbitMQClient) reconnect() {
	r.mu.Lock()
	if r.isReconnecting {
		r.mu.Unlock()
		return
	}
	r.isReconnecting = true
	r.mu.Unlock()

	defer func() {
		r.mu.Lock()
		r.isReconnecting = false
		r.mu.Unlock()
	}()

	for {
		r.logger.Info("Attempting to reconnect to RabbitMQ...")
		err := r.connect(30 * time.Second)
		if err == nil {
			r.logger.Info("Successfully reconnected to RabbitMQ and restored topology!")
			break
		}
		r.logger.Error("Reconnect attempt failed, retrying in 5 seconds...", zap.Error(err))
		time.Sleep(5 * time.Second)
	}
}
