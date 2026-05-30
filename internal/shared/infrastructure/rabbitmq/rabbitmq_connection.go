package rabbitmq

import (
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

func (r *RabbitMQClient) connect() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	url := fmt.Sprintf("amqp://%s:%s@%s:%d/",
		r.cfg.RabbitMQ.User,
		r.cfg.RabbitMQ.Password,
		r.cfg.RabbitMQ.Host,
		r.cfg.RabbitMQ.Port,
	)

	var err error
	for i := 0; i < 5; i++ {
		r.conn, err = amqp.Dial(url)
		if err == nil {
			break
		}
		r.logger.Warn("RabbitMQ connection attempt failed, retrying...",
			zap.Int("attempt", i+1),
			zap.Int("max_attempts", 5),
			zap.Error(err),
		)
		time.Sleep(3 * time.Second)
	}

	if err != nil {
		return fmt.Errorf("could not connect to RabbitMQ broker: %w", err)
	}

	r.ch, err = r.conn.Channel()
	if err != nil {
		_ = r.conn.Close()
		return fmt.Errorf("could not open RabbitMQ channel: %w", err)
	}

	// 1. Khởi tạo Topic Exchange linh hoạt, chuẩn Pub/Sub hướng sự kiện
	err = r.ch.ExchangeDeclare(
		ExchangeName, // exchange name
		ExchangeType, // exchange type ("topic")
		true,         // durable (bền vững)
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

	// 2. Declare queue mặc định cho batch crop
	_, err = r.ch.QueueDeclare(
		"batch_crop_jobs", // queue name
		true,               // durable (bền vững)
		false,              // delete when unused
		false,              // exclusive
		false,              // no-wait
		nil,                // arguments
	)
	if err != nil {
		_ = r.ch.Close()
		_ = r.conn.Close()
		return fmt.Errorf("could not declare batch_crop_jobs queue: %w", err)
	}

	// 3. Bind Queue vào Exchange với routing key tương ứng (Pub/Sub)
	err = r.ch.QueueBind(
		"batch_crop_jobs", // queue name
		"batch_crop_jobs", // routing key
		ExchangeName,      // exchange
		false,
		nil,
	)
	if err != nil {
		_ = r.ch.Close()
		_ = r.conn.Close()
		return fmt.Errorf("could not bind queue to exchange: %w", err)
	}

	r.logger.Info("Successfully connected to RabbitMQ and established binding topology.")

	// 4. Đăng ký lắng nghe NotifyClose để tự động reconnect ngầm khi mất mạng
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
		err := r.connect()
		if err == nil {
			r.logger.Info("Successfully reconnected to RabbitMQ and restored topology!")
			break
		}
		r.logger.Error("Reconnect attempt failed, retrying in 5 seconds...", zap.Error(err))
		time.Sleep(5 * time.Second)
	}
}
