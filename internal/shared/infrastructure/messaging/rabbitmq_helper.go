package messaging

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

// DeclareAndBindQueue is a helper method to declare a queue and bind it to the topic exchange (backward compatible).
func (r *RabbitMQClient) DeclareAndBindQueue(queueName, routingKey string) error {
	return r.DeclareAndBindQueueWithArgs(queueName, routingKey, ExchangeName, nil)
}

// DeclareAndBindQueueWithArgs declares a durable queue with custom args and binds it to the given exchange.
func (r *RabbitMQClient) DeclareAndBindQueueWithArgs(queueName, routingKey, exchange string, args amqp.Table) error {
	if r.ch == nil {
		return fmt.Errorf("RabbitMQ channel is not open")
	}

	// 1. Declare Durable Queue
	_, err := r.ch.QueueDeclare(
		queueName, // queue name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		args,      // arguments (quorum, DLX, etc.)
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue %s: %w", queueName, err)
	}

	// 2. Bind Queue to Exchange with the corresponding routing key
	err = r.ch.QueueBind(
		queueName,  // queue name
		routingKey, // routing key
		exchange,   // exchange
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to bind queue %s to routing key %s: %w", queueName, routingKey, err)
	}

	return nil
}
