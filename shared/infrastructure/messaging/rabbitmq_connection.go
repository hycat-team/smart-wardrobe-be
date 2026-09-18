package messaging

import (
	"context"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

func (r *RabbitMQClient) connect(timeout time.Duration) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	url := amqpURL(r.cfg)

	conn, err := amqp.DialConfig(url, amqp.Config{
		Locale:          "en_US",
		Dial:            amqp.DefaultDial(timeout),
		TLSClientConfig: amqpTLSConfig(r.cfg),
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
	r.publisherCh, err = r.conn.Channel()
	if err != nil {
		_ = r.ch.Close()
		_ = r.conn.Close()
		return fmt.Errorf("could not open RabbitMQ publisher channel: %w", err)
	}
	if err := r.publisherCh.Confirm(false); err != nil {
		r.logger.Warn("Failed to enable publisher confirms", zap.Error(err))
	} else {
		// Mỗi publish trên channel này có đúng 1 confirmation (theo thứ tự);
		// PublishDirect đợi ack trước khi trả nil để OutboxUseCase chỉ MarkPublished
		// khi broker đã thực nhận. publisherMu serialize publish nên confirmation
		// đọc ra luôn thuộc về publish vừa gọi.
		r.confirms = r.publisherCh.NotifyPublish(make(chan amqp.Confirmation, 256))
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
		if r.publisherCh != nil {
			_ = r.publisherCh.Close()
		}
		_ = r.ch.Close()
		_ = r.conn.Close()
		return fmt.Errorf("could not declare RabbitMQ exchange: %w", err)
	}

	// 1b. Declare DLX topic exchange for dead-lettering
	err = r.ch.ExchangeDeclare(
		ExchangeDLXName,
		ExchangeDLXType,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		if r.publisherCh != nil {
			_ = r.publisherCh.Close()
		}
		_ = r.ch.Close()
		_ = r.conn.Close()
		return fmt.Errorf("could not declare DLX exchange: %w", err)
	}

	// 2. Declare and bind primary topology via registry
	for _, b := range Topology {
		for _, rk := range b.RoutingKeys {
			if err := r.DeclareAndBindQueueWithArgs(b.Queue, rk, b.Exchange, b.Args); err != nil {
				if r.publisherCh != nil {
					_ = r.publisherCh.Close()
				}
				_ = r.ch.Close()
				_ = r.conn.Close()
				return err
			}
		}
	}

	// 3. Declare and bind DLQ topology
	for _, b := range DLQTopology {
		for _, rk := range b.RoutingKeys {
			if err := r.DeclareAndBindQueueWithArgs(b.Queue, rk, b.Exchange, b.Args); err != nil {
				if r.publisherCh != nil {
					_ = r.publisherCh.Close()
				}
				_ = r.ch.Close()
				_ = r.conn.Close()
				return err
			}
		}
	}

	r.logger.Info("Successfully connected to RabbitMQ and established binding topology.")

	// 4. Register to listen to NotifyClose for auto-reconnection in the background upon disconnection
	errChan := make(chan *amqp.Error, 1)
	r.conn.NotifyClose(errChan)
	ctx := context.Background()
	go r.handleReconnect(ctx, errChan)

	return nil
}

func (r *RabbitMQClient) handleReconnect(ctx context.Context, errChan chan *amqp.Error) {
	select {
	case <-ctx.Done():
		return
	case reason, ok := <-errChan:
		if !ok || reason == nil {
			return
		}
		r.logger.Warn("RabbitMQ connection disrupted. Initiating auto-reconnection...", zap.Error(reason))
		r.reconnect(ctx)
	}
}

func (r *RabbitMQClient) reconnect(ctx context.Context) {
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

	backoff := 1 * time.Second
	maxBackoff := 30 * time.Second
	for attempt := 0; ; attempt++ {
		select {
		case <-ctx.Done():
			r.logger.Info("RabbitMQ reconnect loop cancelled")
			return
		default:
		}

		r.logger.Info("Attempting to reconnect to RabbitMQ...", zap.Int("attempt", attempt+1))
		incReconnect()
		err := r.connect(30 * time.Second)
		if err == nil {
			r.logger.Info("Successfully reconnected to RabbitMQ and restored topology!")
			r.reRegisterConsumers()
			break
		}
		r.logger.Error("Reconnect attempt failed, retrying...", zap.Error(err), zap.Duration("nextBackoff", backoff))

		select {
		case <-ctx.Done():
			r.logger.Info("RabbitMQ reconnect loop cancelled during backoff")
			return
		case <-time.After(backoff):
		}

		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}
