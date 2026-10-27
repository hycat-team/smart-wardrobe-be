package messaging

import (
	"strings"
	"testing"

	"smart-wardrobe-be/shared/application/constants/eventconstants"
)

// 025 T013 (FR-092/FR-093): identity.user.created must reach EXACTLY ONE queue.
//
// The number matters more than the identity. A grant is not idempotent by design —
// it is made idempotent by the unique claim row inside the consumer — so two
// bindings would be two concurrent grant attempts competing for the same budget,
// and the second one would only ever look like a quota bug. The queue that owns
// the topic is therefore pinned here, by name.
func TestTopologyBindsUserCreatedToExactlyOneQueue(t *testing.T) {
	owners := bindingsRoutingTo(Topology, eventconstants.TopicUserCreated)

	if len(owners) != 1 {
		t.Fatalf("%s is bound by %d queues (%s), want exactly 1: a second consumer would race the first for the same campaign budget",
			eventconstants.TopicUserCreated, len(owners), queueNames(owners))
	}
	if owners[0].Queue != eventconstants.QueueCampaignGrantUserCreated {
		t.Errorf("%s is bound to queue %q, want %q",
			eventconstants.TopicUserCreated, owners[0].Queue, eventconstants.QueueCampaignGrantUserCreated)
	}
}

// TestCampaignGrantQueueDeliveryContract guards the R12 delivery decision: quorum
// with x-delivery-limit 5, prefetch 20, concurrency 1, plus the matching dead
// letter queue.
//
// Both halves of that decision are needed. Quorum alone dead-letters a transient
// failure on the first attempt; the delivery limit alone never fires on a classic
// queue. Either one missing means a grant that would have succeeded on a retry
// loses its account instead.
func TestCampaignGrantQueueDeliveryContract(t *testing.T) {
	owners := bindingsRoutingTo(Topology, eventconstants.TopicUserCreated)
	if len(owners) != 1 {
		t.Fatalf("campaign grant binding not found (%d owners), cannot check its delivery contract", len(owners))
	}
	binding := owners[0]

	if got := binding.Args["x-queue-type"]; got != "quorum" {
		t.Errorf("x-queue-type = %v, want quorum (a redelivered grant must be bounded)", got)
	}
	if got, ok := binding.Args["x-delivery-limit"]; !ok || got != 5 {
		t.Errorf("x-delivery-limit = %v, want 5", got)
	}
	if got := binding.Args["x-dead-letter-exchange"]; got != ExchangeDLXName {
		t.Errorf("x-dead-letter-exchange = %v, want %s", got, ExchangeDLXName)
	}
	if got := binding.Args["x-dead-letter-routing-key"]; got != eventconstants.QueueCampaignGrantUserCreated+".dlq" {
		t.Errorf("x-dead-letter-routing-key = %v, want %s.dlq", got, eventconstants.QueueCampaignGrantUserCreated)
	}
	if binding.Prefetch != 20 || binding.Concurrency != 1 {
		t.Errorf("prefetch/concurrency = %d/%d, want 20/1", binding.Prefetch, binding.Concurrency)
	}

	// queueArgs routes to <queue>.dlq, so a DLQ binding has to exist or rejected
	// messages are dropped by the broker with nothing to inspect.
	dlqName := eventconstants.QueueCampaignGrantUserCreated + ".dlq"
	deadLetters := bindingsRoutingTo(DLQTopology, dlqName)
	if len(deadLetters) != 1 {
		t.Fatalf("%s is bound %d times in DLQTopology, want 1", dlqName, len(deadLetters))
	}
	if got := deadLetters[0].Exchange; got != ExchangeDLXName {
		t.Errorf("%s bound to exchange %q, want %s", dlqName, got, ExchangeDLXName)
	}
	if _, ok := deadLetters[0].Args["x-queue-type"]; !ok {
		t.Errorf("%s has no x-queue-type argument", dlqName)
	}
}

// TestCampaignGrantQueueIsNotBoundToAnotherTopic keeps the new queue narrow: it is
// a grant queue, not a general identity consumer, so a second routing key on it
// would silently make every future identity event a grant input.
func TestCampaignGrantQueueIsNotBoundToAnotherTopic(t *testing.T) {
	for _, binding := range Topology {
		if binding.Queue != eventconstants.QueueCampaignGrantUserCreated {
			continue
		}
		for _, key := range binding.RoutingKeys {
			if key != eventconstants.TopicUserCreated {
				t.Errorf("queue %s is also bound to %q; only %s may reach a grant queue",
					binding.Queue, key, eventconstants.TopicUserCreated)
			}
		}
		return
	}
	t.Fatalf("queue %s is missing from Topology", eventconstants.QueueCampaignGrantUserCreated)
}

// TestStyleLearningQueueDeliveryContract guards the 033 T017 delivery decision
// for the outfit-save signal: quorum with x-delivery-limit 5 plus a matching
// dead letter queue.
//
// The consumer is idempotent by signal id (ON CONFLICT (id) DO NOTHING), so the
// cost of a redelivery is a no-op — but only if a redelivery happens at all. On
// a classic queue a transient database failure is redelivered forever, while
// quorum alone dead-letters it on the first attempt; both halves are required,
// exactly as for the plan-changed queue.
func TestStyleLearningQueueDeliveryContract(t *testing.T) {
	// The broker name is pinned here on purpose: renaming the constant would
	// leave the old `style_learning_queue` declared on the broker and the new
	// one empty, and nothing else in the code would notice.
	if eventconstants.QueueStyleLearning != "style_learning_queue" {
		t.Fatalf("QueueStyleLearning = %q, want style_learning_queue", eventconstants.QueueStyleLearning)
	}

	owners := bindingsRoutingTo(Topology, eventconstants.TopicOutfitSaved)
	if len(owners) != 1 {
		t.Fatalf("%s is bound by %d queues (%s), want exactly 1: a second consumer would apply the same "+
			"signal twice", eventconstants.TopicOutfitSaved, len(owners), queueNames(owners))
	}
	binding := owners[0]
	if binding.Queue != eventconstants.QueueStyleLearning {
		t.Fatalf("%s is bound to queue %q, want %q",
			eventconstants.TopicOutfitSaved, binding.Queue, eventconstants.QueueStyleLearning)
	}
	if got := binding.Args["x-queue-type"]; got != QueueTypeQuorum {
		t.Errorf("x-queue-type = %v, want %s (a redelivered signal must be bounded)", got, QueueTypeQuorum)
	}
	if got, ok := binding.Args["x-delivery-limit"]; !ok || got != 5 {
		t.Errorf("x-delivery-limit = %v, want 5", got)
	}
	if got := binding.Args["x-dead-letter-exchange"]; got != ExchangeDLXName {
		t.Errorf("x-dead-letter-exchange = %v, want %s", got, ExchangeDLXName)
	}
	if got := binding.Args["x-dead-letter-routing-key"]; got != eventconstants.QueueStyleLearning+".dlq" {
		t.Errorf("x-dead-letter-routing-key = %v, want %s.dlq", got, eventconstants.QueueStyleLearning)
	}
	if binding.Prefetch != 20 || binding.Concurrency != 1 {
		t.Errorf("prefetch/concurrency = %d/%d, want 20/1", binding.Prefetch, binding.Concurrency)
	}

	// queueArgs routes to <queue>.dlq, so the DLQ binding has to exist or a
	// rejected message is dropped by the broker with nothing to inspect.
	dlqName := eventconstants.QueueStyleLearning + ".dlq"
	deadLetters := bindingsRoutingTo(DLQTopology, dlqName)
	if len(deadLetters) != 1 {
		t.Fatalf("%s is bound %d times in DLQTopology, want 1", dlqName, len(deadLetters))
	}
	if got := deadLetters[0].Exchange; got != ExchangeDLXName {
		t.Errorf("%s bound to exchange %q, want %s", dlqName, got, ExchangeDLXName)
	}
	if got := deadLetters[0].Args["x-queue-type"]; got != QueueTypeQuorum {
		t.Errorf("%s x-queue-type = %v, want %s: a dead-letter queue for a quorum queue must be quorum",
			dlqName, got, QueueTypeQuorum)
	}

	// The topology drift tool reads this map; it must agree with the binding.
	if got := DesiredQueueTypes()[eventconstants.QueueStyleLearning]; got != QueueTypeQuorum {
		t.Errorf("DesiredQueueTypes()[%s] = %q, want %q",
			eventconstants.QueueStyleLearning, got, QueueTypeQuorum)
	}
}

// bindingsRoutingTo returns every binding that routes to key.
func bindingsRoutingTo(bindings []Binding, key string) []Binding {
	var matches []Binding
	for _, binding := range bindings {
		for _, routingKey := range binding.RoutingKeys {
			if routingKey == key {
				matches = append(matches, binding)
				break
			}
		}
	}
	return matches
}

func queueNames(bindings []Binding) string {
	names := make([]string, 0, len(bindings))
	for _, binding := range bindings {
		names = append(names, binding.Queue)
	}
	return strings.Join(names, ", ")
}
