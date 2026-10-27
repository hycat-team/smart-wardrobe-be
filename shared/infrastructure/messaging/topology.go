package messaging

import (
	"smart-wardrobe-be/shared/application/constants/eventconstants"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	ExchangeDLXName = "smart_wardrobe_exchange.dlx"
	ExchangeDLXType = "topic"

	// QueueTypeClassic is the queue type RabbitMQ assigns when a queue is
	// declared without x-queue-type.
	QueueTypeClassic = "classic"
	// QueueTypeQuorum is the queue type declared via x-queue-type=quorum.
	QueueTypeQuorum = "quorum"
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
		Queue:       eventconstants.QueueElasticsearchSync,
		RoutingKeys: []string{RoutingKeyElasticsearchSyncWildcard},
		Exchange:    ExchangeName,
		Args:        queueArgs(eventconstants.QueueElasticsearchSync, false),
		Prefetch:    20,
		Concurrency: 1,
	},
	{
		Queue:       eventconstants.QueueFashionInit,
		RoutingKeys: []string{RoutingKeyFashionInit},
		Exchange:    ExchangeName,
		Args:        queueArgs(eventconstants.QueueFashionInit, false),
		Prefetch:    20,
		Concurrency: 1,
	},
	{
		Queue:       eventconstants.QueueFashionAnalyze,
		RoutingKeys: []string{RoutingKeyFashionAnalyze},
		Exchange:    ExchangeName,
		Args:        queueArgs(eventconstants.QueueFashionAnalyze, true),
		Prefetch:    20,
		Concurrency: 3,
	},
	{
		Queue:       eventconstants.QueueSubscriptionPlanChanged,
		RoutingKeys: []string{RoutingKeySubscriptionPlanChanged},
		Exchange:    ExchangeName,
		// 024 T053 / FR-017: quorum with x-delivery-limit 5. On a classic queue
		// a RetryableError is redelivered forever, and a message that would
		// succeed on the second attempt still never gets one; on a quorum queue
		// it is retried five times and then dead-lettered. Both halves are
		// required — quorum alone still dead-letters a transient failure on the
		// first attempt, and RetryableError alone spins forever.
		Args:        queueArgs(eventconstants.QueueSubscriptionPlanChanged, true),
		Prefetch:    20,
		Concurrency: 1,
	},
	{
		Queue:       eventconstants.QueueNotificationEmail,
		RoutingKeys: []string{RoutingKeyNotificationEmail},
		Exchange:    ExchangeName,
		Args:        queueArgs(eventconstants.QueueNotificationEmail, false),
		Prefetch:    20,
		Concurrency: 1,
	},
	{
		Queue:       eventconstants.QueueAuditLog,
		RoutingKeys: []string{RoutingKeyAuditLog},
		Exchange:    ExchangeName,
		Args:        queueArgs(eventconstants.QueueAuditLog, false),
		Prefetch:    20,
		Concurrency: 1,
	},
	{
		Queue:       eventconstants.QueueWardrobeFashionAnalyzed,
		RoutingKeys: []string{RoutingKeyFashionAnalyzedWardrobe, RoutingKeyFashionFailedWardrobe, RoutingKeyFashionProcessingWardrobe},
		Exchange:    ExchangeName,
		Args:        queueArgs(eventconstants.QueueWardrobeFashionAnalyzed, true),
		Prefetch:    20,
		Concurrency: 1,
	},
	{
		Queue:       eventconstants.QueueBrandFashionAnalyzed,
		RoutingKeys: []string{RoutingKeyFashionAnalyzedBrand, RoutingKeyFashionFailedBrand, RoutingKeyFashionProcessingBrand},
		Exchange:    ExchangeName,
		Args:        queueArgs(eventconstants.QueueBrandFashionAnalyzed, true),
		Prefetch:    20,
		Concurrency: 1,
	},
	{
		Queue:       eventconstants.QueueWardrobeCleanup,
		RoutingKeys: []string{RoutingKeyWardrobeDeleted},
		Exchange:    ExchangeName,
		Args:        queueArgs(eventconstants.QueueWardrobeCleanup, false),
		Prefetch:    20,
		Concurrency: 1,
	},
	{
		Queue:       eventconstants.QueueWardrobeProcessingExhausted,
		RoutingKeys: []string{RoutingKeyFashionProcessingExhausted},
		Exchange:    ExchangeName,
		Args:        queueArgs(eventconstants.QueueWardrobeProcessingExhausted, false),
		Prefetch:    20,
		Concurrency: 1,
	},
	{
		Queue:       eventconstants.QueueBrandProcessingExhausted,
		RoutingKeys: []string{RoutingKeyFashionProcessingExhausted},
		Exchange:    ExchangeName,
		Args:        queueArgs(eventconstants.QueueBrandProcessingExhausted, false),
		Prefetch:    20,
		Concurrency: 1,
	},
	{
		Queue:       eventconstants.QueueElasticsearchProcessingExhausted,
		RoutingKeys: []string{RoutingKeyFashionProcessingExhausted},
		Exchange:    ExchangeName,
		Args:        queueArgs(eventconstants.QueueElasticsearchProcessingExhausted, false),
		Prefetch:    20,
		Concurrency: 1,
	},
	{
		Queue:       eventconstants.QueueWardrobeFashionBind,
		RoutingKeys: []string{RoutingKeyFashionItemCreatedWardrobe},
		Exchange:    ExchangeName,
		Args:        queueArgs(eventconstants.QueueWardrobeFashionBind, false),
		Prefetch:    20,
		Concurrency: 1,
	},
	{
		Queue:       eventconstants.QueueBrandFashionBind,
		RoutingKeys: []string{RoutingKeyFashionItemCreatedBrand},
		Exchange:    ExchangeName,
		Args:        queueArgs(eventconstants.QueueBrandFashionBind, false),
		Prefetch:    20,
		Concurrency: 1,
	},
	{
		Queue:       eventconstants.QueueCommunityOutfitCascade,
		RoutingKeys: []string{RoutingKeyOutfitDeleted},
		Exchange:    ExchangeName,
		Args:        queueArgs(eventconstants.QueueCommunityOutfitCascade, false),
		Prefetch:    20,
		Concurrency: 1,
	},
	{
		Queue:       eventconstants.QueueCommunityUserStatus,
		RoutingKeys: []string{RoutingKeyUserStatusChanged},
		Exchange:    ExchangeName,
		Args:        queueArgs(eventconstants.QueueCommunityUserStatus, false),
		Prefetch:    20,
		Concurrency: 1,
	},
	{
		Queue:       eventconstants.QueueStyleLearning,
		RoutingKeys: []string{RoutingKeyOutfitSaved},
		Exchange:    ExchangeName,
		// 033 T017: quorum with x-delivery-limit 5, same reasoning as
		// QueueSubscriptionPlanChanged above. The consumer is idempotent by
		// signal id, so a transient database failure is retried and a poisoned
		// message is dead-lettered after five attempts instead of looping.
		Args:        queueArgs(eventconstants.QueueStyleLearning, true),
		Prefetch:    20,
		Concurrency: 1,
	},
	{
		Queue:       eventconstants.QueueCampaignGrantUserCreated,
		RoutingKeys: []string{RoutingKeyUserCreated},
		Exchange:    ExchangeName,
		// 025 T004 / FR-093: quorum with x-delivery-limit 5, same reasoning as
		// QueueSubscriptionPlanChanged above. A grant that failed for a transient
		// reason is redelivered up to five times and then dead-lettered; on a
		// classic queue it would be redelivered forever, and a message that would
		// succeed on the second attempt never gets that second attempt.
		Args:        queueArgs(eventconstants.QueueCampaignGrantUserCreated, true),
		Prefetch:    20,
		Concurrency: 1,
	},
}

// DLQTopology defines dead-letter queues bound to the DLX exchange.
var DLQTopology = []Binding{
	{Queue: eventconstants.QueueElasticsearchSync + ".dlq", RoutingKeys: []string{eventconstants.QueueElasticsearchSync + ".dlq"}, Exchange: ExchangeDLXName, Args: amqp.Table{"x-queue-type": "quorum"}, Prefetch: 20, Concurrency: 1},
	{Queue: eventconstants.QueueFashionInit + ".dlq", RoutingKeys: []string{eventconstants.QueueFashionInit + ".dlq"}, Exchange: ExchangeDLXName, Args: amqp.Table{"x-queue-type": "quorum"}, Prefetch: 20, Concurrency: 1},
	{Queue: eventconstants.QueueFashionAnalyze + ".dlq", RoutingKeys: []string{eventconstants.QueueFashionAnalyze + ".dlq"}, Exchange: ExchangeDLXName, Args: amqp.Table{"x-queue-type": "quorum"}, Prefetch: 20, Concurrency: 1},
	{Queue: eventconstants.QueueSubscriptionPlanChanged + ".dlq", RoutingKeys: []string{eventconstants.QueueSubscriptionPlanChanged + ".dlq"}, Exchange: ExchangeDLXName, Args: amqp.Table{"x-queue-type": "quorum"}, Prefetch: 20, Concurrency: 1},
	{Queue: eventconstants.QueueNotificationEmail + ".dlq", RoutingKeys: []string{eventconstants.QueueNotificationEmail + ".dlq"}, Exchange: ExchangeDLXName, Args: amqp.Table{"x-queue-type": "quorum"}, Prefetch: 20, Concurrency: 1},
	{Queue: eventconstants.QueueAuditLog + ".dlq", RoutingKeys: []string{eventconstants.QueueAuditLog + ".dlq"}, Exchange: ExchangeDLXName, Args: amqp.Table{"x-queue-type": "quorum"}, Prefetch: 20, Concurrency: 1},
	{Queue: eventconstants.QueueWardrobeFashionAnalyzed + ".dlq", RoutingKeys: []string{eventconstants.QueueWardrobeFashionAnalyzed + ".dlq"}, Exchange: ExchangeDLXName, Args: amqp.Table{"x-queue-type": "quorum"}, Prefetch: 20, Concurrency: 1},
	{Queue: eventconstants.QueueBrandFashionAnalyzed + ".dlq", RoutingKeys: []string{eventconstants.QueueBrandFashionAnalyzed + ".dlq"}, Exchange: ExchangeDLXName, Args: amqp.Table{"x-queue-type": "quorum"}, Prefetch: 20, Concurrency: 1},
	{Queue: eventconstants.QueueWardrobeCleanup + ".dlq", RoutingKeys: []string{eventconstants.QueueWardrobeCleanup + ".dlq"}, Exchange: ExchangeDLXName, Args: amqp.Table{"x-queue-type": "quorum"}, Prefetch: 20, Concurrency: 1},
	{Queue: eventconstants.QueueWardrobeProcessingExhausted + ".dlq", RoutingKeys: []string{eventconstants.QueueWardrobeProcessingExhausted + ".dlq"}, Exchange: ExchangeDLXName, Args: amqp.Table{"x-queue-type": "quorum"}, Prefetch: 20, Concurrency: 1},
	{Queue: eventconstants.QueueBrandProcessingExhausted + ".dlq", RoutingKeys: []string{eventconstants.QueueBrandProcessingExhausted + ".dlq"}, Exchange: ExchangeDLXName, Args: amqp.Table{"x-queue-type": "quorum"}, Prefetch: 20, Concurrency: 1},
	{Queue: eventconstants.QueueElasticsearchProcessingExhausted + ".dlq", RoutingKeys: []string{eventconstants.QueueElasticsearchProcessingExhausted + ".dlq"}, Exchange: ExchangeDLXName, Args: amqp.Table{"x-queue-type": "quorum"}, Prefetch: 20, Concurrency: 1},
	{Queue: eventconstants.QueueWardrobeFashionBind + ".dlq", RoutingKeys: []string{eventconstants.QueueWardrobeFashionBind + ".dlq"}, Exchange: ExchangeDLXName, Args: amqp.Table{"x-queue-type": "quorum"}, Prefetch: 20, Concurrency: 1},
	{Queue: eventconstants.QueueBrandFashionBind + ".dlq", RoutingKeys: []string{eventconstants.QueueBrandFashionBind + ".dlq"}, Exchange: ExchangeDLXName, Args: amqp.Table{"x-queue-type": "quorum"}, Prefetch: 20, Concurrency: 1},
	{Queue: eventconstants.QueueCommunityOutfitCascade + ".dlq", RoutingKeys: []string{eventconstants.QueueCommunityOutfitCascade + ".dlq"}, Exchange: ExchangeDLXName, Args: amqp.Table{"x-queue-type": "quorum"}, Prefetch: 20, Concurrency: 1},
	{Queue: eventconstants.QueueCommunityUserStatus + ".dlq", RoutingKeys: []string{eventconstants.QueueCommunityUserStatus + ".dlq"}, Exchange: ExchangeDLXName, Args: amqp.Table{"x-queue-type": "quorum"}, Prefetch: 20, Concurrency: 1},
	{Queue: eventconstants.QueueStyleLearning + ".dlq", RoutingKeys: []string{eventconstants.QueueStyleLearning + ".dlq"}, Exchange: ExchangeDLXName, Args: amqp.Table{"x-queue-type": "quorum"}, Prefetch: 20, Concurrency: 1},
	{Queue: eventconstants.QueueCampaignGrantUserCreated + ".dlq", RoutingKeys: []string{eventconstants.QueueCampaignGrantUserCreated + ".dlq"}, Exchange: ExchangeDLXName, Args: amqp.Table{"x-queue-type": "quorum"}, Prefetch: 20, Concurrency: 1},
}

// DesiredQueueTypes returns, per queue, the queue type this registry declares —
// "quorum" when Args carries x-queue-type=quorum, otherwise "classic" (the
// RabbitMQ default when the argument is absent).
//
// RabbitMQ cannot change x-queue-type on an existing queue, so Topology and the
// broker are two independent sources of truth: a registry edited to declare
// quorum against a broker that still holds a classic queue makes every service
// fail at startup with 406 PRECONDITION_FAILED. The rabbitmq-topology tool reads
// this map to detect that drift; it must never keep its own copy of the list.
func DesiredQueueTypes() map[string]string {
	desired := make(map[string]string, len(Topology)+len(DLQTopology))
	for _, b := range append(append([]Binding{}, Topology...), DLQTopology...) {
		if t, ok := b.Args["x-queue-type"]; ok {
			desired[b.Queue] = t.(string)
		} else {
			desired[b.Queue] = QueueTypeClassic
		}
	}
	return desired
}
