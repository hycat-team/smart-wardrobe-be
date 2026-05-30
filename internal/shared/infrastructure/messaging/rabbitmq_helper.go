package messaging

import (
	"fmt"
)

// DeclareAndBindQueue is a helper method to declare a queue and bind it to the topic exchange
func (r *RabbitMQClient) DeclareAndBindQueue(queueName, routingKey string) error {
	if r.ch == nil {
		return fmt.Errorf("RabbitMQ channel is not open")
	}

	// 1. Khai báo Queue bền vững (Durable)
	_, err := r.ch.QueueDeclare(
		queueName, // queue name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue %s: %w", queueName, err)
	}

	// 2. Bind Queue vào Exchange với routing key tương ứng
	err = r.ch.QueueBind(
		queueName,    // queue name
		routingKey,   // routing key
		ExchangeName, // exchange
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to bind queue %s to routing key %s: %w", queueName, routingKey, err)
	}

	return nil
}
