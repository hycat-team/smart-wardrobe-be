package usecase

import (
	"context"
	"time"
)

// IDeliveryDeduplicator remembers which outbox rows have already been delivered.
//
// It exists because RabbitMQ delivers at least once: a consumer that dies after
// SMTP accepted the message but before the ack leaves the broker with no choice
// but to redeliver, and a customer who receives the same receipt twice reasonably
// concludes the system charged them twice.
type IDeliveryDeduplicator interface {
	// Seen reports whether this outbox row has already been delivered.
	Seen(ctx context.Context, key string) (bool, error)
	// Mark records a successful delivery for the given retention.
	Mark(ctx context.Context, key string, ttl time.Duration) error
}
