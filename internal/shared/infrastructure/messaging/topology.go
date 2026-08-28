package messaging

import amqp "github.com/rabbitmq/amqp091-go"

const (
	ExchangeDLXName = "smart_wardrobe_exchange.dlx"
	ExchangeDLXType = "topic"
)

// Binding defines a queue binding to an exchange with routing keys and queue arguments.
type Binding struct {
	Queue       string
	RoutingKeys []string
	Exchange    string
	Args        amqp.Table
	Prefetch    int
	Concurrency int
}

func queueArgs(queue string, isQuorum bool) amqp.Table {
	args := amqp.Table{
		"x-dead-letter-exchange":    ExchangeDLXName,
		"x-dead-letter-routing-key": queue + ".dlq",
	}
	if isQuorum {
		args["x-queue-type"] = "quorum"
		args["x-delivery-limit"] = 5
	}
	return args
}

// Topology is the central registry for all primary queue bindings.
// Adding a new queue only requires appending an entry here.
var Topology = []Binding{
	{
		Queue:       QueueElasticsearchSync,
		RoutingKeys: []string{RoutingKeyElasticsearchSyncWildcard},
		Exchange:    ExchangeName,
		Args:        queueArgs(QueueElasticsearchSync, false),
		Prefetch:    20,
		Concurrency: 1,
	},
	{
		Queue:       QueueFashionInit,
		RoutingKeys: []string{RoutingKeyFashionInit},
		Exchange:    ExchangeName,
		Args:        queueArgs(QueueFashionInit, false),
		Prefetch:    20,
		Concurrency: 1,
	},
	{
		Queue:       QueueFashionAnalyze,
		RoutingKeys: []string{RoutingKeyFashionAnalyze},
		Exchange:    ExchangeName,
		Args:        queueArgs(QueueFashionAnalyze, true),
		Prefetch:    20,
		Concurrency: 3,
	},
	{
		Queue:       QueueSubscriptionPlanChanged,
		RoutingKeys: []string{RoutingKeySubscriptionPlanChanged},
		Exchange:    ExchangeName,
		Args:        queueArgs(QueueSubscriptionPlanChanged, false),
		Prefetch:    20,
		Concurrency: 1,
	},
	{
		Queue:       QueueNotificationEmail,
		RoutingKeys: []string{RoutingKeyNotificationEmail},
		Exchange:    ExchangeName,
		Args:        queueArgs(QueueNotificationEmail, false),
		Prefetch:    20,
		Concurrency: 1,
	},
	{
		Queue:       QueueAuditLog,
		RoutingKeys: []string{RoutingKeyAuditLog},
		Exchange:    ExchangeName,
		Args:        queueArgs(QueueAuditLog, false),
		Prefetch:    20,
		Concurrency: 1,
	},
	{
		Queue:       QueueWardrobeFashionAnalyzed,
		RoutingKeys: []string{RoutingKeyFashionAnalyzedWardrobe, RoutingKeyFashionFailedWardrobe},
		Exchange:    ExchangeName,
		Args:        queueArgs(QueueWardrobeFashionAnalyzed, true),
		Prefetch:    20,
		Concurrency: 1,
	},
	{
		Queue:       QueueBrandFashionAnalyzed,
		RoutingKeys: []string{RoutingKeyFashionAnalyzedBrand, RoutingKeyFashionFailedBrand},
		Exchange:    ExchangeName,
		Args:        queueArgs(QueueBrandFashionAnalyzed, true),
		Prefetch:    20,
		Concurrency: 1,
	},
	{
		Queue:       QueueWardrobeCleanup,
		RoutingKeys: []string{RoutingKeyWardrobeDeleted},
		Exchange:    ExchangeName,
		Args:        queueArgs(QueueWardrobeCleanup, false),
		Prefetch:    20,
		Concurrency: 1,
	},
	{
		Queue:       QueueWardrobeProcessingExhausted,
		RoutingKeys: []string{RoutingKeyFashionProcessingExhausted},
		Exchange:    ExchangeName,
		Args:        queueArgs(QueueWardrobeProcessingExhausted, false),
		Prefetch:    20,
		Concurrency: 1,
	},
	{
		Queue:       QueueElasticsearchProcessingExhausted,
		RoutingKeys: []string{RoutingKeyFashionProcessingExhausted},
		Exchange:    ExchangeName,
		Args:        queueArgs(QueueElasticsearchProcessingExhausted, false),
		Prefetch:    20,
		Concurrency: 1,
	},
	{
		Queue:       QueueWardrobeFashionBind,
		RoutingKeys: []string{RoutingKeyFashionItemCreatedWardrobe},
		Exchange:    ExchangeName,
		Args:        queueArgs(QueueWardrobeFashionBind, false),
		Prefetch:    20,
		Concurrency: 1,
	},
	{
		Queue:       QueueBrandFashionBind,
		RoutingKeys: []string{RoutingKeyFashionItemCreatedBrand},
		Exchange:    ExchangeName,
		Args:        queueArgs(QueueBrandFashionBind, false),
		Prefetch:    20,
		Concurrency: 1,
	},
}

// DLQTopology defines dead-letter queues bound to the DLX exchange.
var DLQTopology = []Binding{
	{Queue: QueueElasticsearchSync + ".dlq", RoutingKeys: []string{QueueElasticsearchSync + ".dlq"}, Exchange: ExchangeDLXName, Args: amqp.Table{"x-queue-type": "quorum"}, Prefetch: 20, Concurrency: 1},
	{Queue: QueueFashionInit + ".dlq", RoutingKeys: []string{QueueFashionInit + ".dlq"}, Exchange: ExchangeDLXName, Args: amqp.Table{"x-queue-type": "quorum"}, Prefetch: 20, Concurrency: 1},
	{Queue: QueueFashionAnalyze + ".dlq", RoutingKeys: []string{QueueFashionAnalyze + ".dlq"}, Exchange: ExchangeDLXName, Args: amqp.Table{"x-queue-type": "quorum"}, Prefetch: 20, Concurrency: 1},
	{Queue: QueueSubscriptionPlanChanged + ".dlq", RoutingKeys: []string{QueueSubscriptionPlanChanged + ".dlq"}, Exchange: ExchangeDLXName, Args: amqp.Table{"x-queue-type": "quorum"}, Prefetch: 20, Concurrency: 1},
	{Queue: QueueNotificationEmail + ".dlq", RoutingKeys: []string{QueueNotificationEmail + ".dlq"}, Exchange: ExchangeDLXName, Args: amqp.Table{"x-queue-type": "quorum"}, Prefetch: 20, Concurrency: 1},
	{Queue: QueueAuditLog + ".dlq", RoutingKeys: []string{QueueAuditLog + ".dlq"}, Exchange: ExchangeDLXName, Args: amqp.Table{"x-queue-type": "quorum"}, Prefetch: 20, Concurrency: 1},
	{Queue: QueueWardrobeFashionAnalyzed + ".dlq", RoutingKeys: []string{QueueWardrobeFashionAnalyzed + ".dlq"}, Exchange: ExchangeDLXName, Args: amqp.Table{"x-queue-type": "quorum"}, Prefetch: 20, Concurrency: 1},
	{Queue: QueueBrandFashionAnalyzed + ".dlq", RoutingKeys: []string{QueueBrandFashionAnalyzed + ".dlq"}, Exchange: ExchangeDLXName, Args: amqp.Table{"x-queue-type": "quorum"}, Prefetch: 20, Concurrency: 1},
	{Queue: QueueWardrobeCleanup + ".dlq", RoutingKeys: []string{QueueWardrobeCleanup + ".dlq"}, Exchange: ExchangeDLXName, Args: amqp.Table{"x-queue-type": "quorum"}, Prefetch: 20, Concurrency: 1},
	{Queue: QueueWardrobeProcessingExhausted + ".dlq", RoutingKeys: []string{QueueWardrobeProcessingExhausted + ".dlq"}, Exchange: ExchangeDLXName, Args: amqp.Table{"x-queue-type": "quorum"}, Prefetch: 20, Concurrency: 1},
	{Queue: QueueElasticsearchProcessingExhausted + ".dlq", RoutingKeys: []string{QueueElasticsearchProcessingExhausted + ".dlq"}, Exchange: ExchangeDLXName, Args: amqp.Table{"x-queue-type": "quorum"}, Prefetch: 20, Concurrency: 1},
	{Queue: QueueWardrobeFashionBind + ".dlq", RoutingKeys: []string{QueueWardrobeFashionBind + ".dlq"}, Exchange: ExchangeDLXName, Args: amqp.Table{"x-queue-type": "quorum"}, Prefetch: 20, Concurrency: 1},
	{Queue: QueueBrandFashionBind + ".dlq", RoutingKeys: []string{QueueBrandFashionBind + ".dlq"}, Exchange: ExchangeDLXName, Args: amqp.Table{"x-queue-type": "quorum"}, Prefetch: 20, Concurrency: 1},
}
