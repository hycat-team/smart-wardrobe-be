package event

import "context"

type IEventPublisher interface {
	Publish(ctx context.Context, topic string, payload any) error
}

// IRawMessagePublisher publishes pre-marshaled payloads directly to the
// broker, bypassing outbox durability. Implemented by the messaging client;
// application code depends on this port instead of the infrastructure client
// (feature 017).
type IRawMessagePublisher interface {
	PublishDirect(ctx context.Context, topic string, payload []byte, headers map[string]any) error
}

// IOutboxGaugeProbe refreshes operational gauges (queue depth, dlq depth)
// after the caller supplies the pending outbox count. Implemented by the
// messaging infrastructure.
type IOutboxGaugeProbe interface {
	ProbeGauges(ctx context.Context, pending int64)
}
