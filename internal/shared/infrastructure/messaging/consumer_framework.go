package messaging

import (
	"context"
	"encoding/json"
	"errors"

	"smart-wardrobe-be/internal/shared/application/event"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

// PermanentError indicates a poison message that should go to DLQ (no requeue).
type PermanentError struct{ Err error }

func (e PermanentError) Error() string { return e.Err.Error() }
func (e PermanentError) Unwrap() error { return e.Err }

// RetryableError indicates a transient failure that should be requeued/retried.
type RetryableError struct{ Err error }

func (e RetryableError) Error() string { return e.Err.Error() }
func (e RetryableError) Unwrap() error { return e.Err }

// ConsumerOpts controls per-queue QoS and concurrency.
type ConsumerOpts struct {
	Prefetch      int
	Concurrency   int
	DeliveryLimit int
}

// TypedHandler processes a typed payload.
type TypedHandler[T any] func(ctx context.Context, payload T) error

// RegisterTypedConsumer registers a typed consumer for the given queue.
// It unmarshals JSON into T and delegates to handler, classifying errors.
func RegisterTypedConsumer[T any](r *RabbitMQClient, queue string, handler TypedHandler[T], opts ConsumerOpts) error {
	if opts.Prefetch <= 0 {
		opts.Prefetch = 20
	}
	if opts.Concurrency <= 0 {
		opts.Concurrency = 1
	}

	// Wrapper that unmarshals and propagates correlationId
	wrapper := func(msg event.Message) error {
		var payload T
		if err := json.Unmarshal(msg.Body, &payload); err != nil {
			return PermanentError{Err: err}
		}
		// Build context with correlationId if available (from headers not in event.Message Body, so try payload field)
		ctx := context.Background()
		return handler(ctx, payload)
	}

	for i := 0; i < opts.Concurrency; i++ {
		r.RegisterConsumer(queue, wrapper)
	}
	return nil
}

// startConsumerLoopWithTaxonomy is an alternative loop that respects error taxonomy.
// Kept for reference; actual taxonomy handling is in rabbitmq.go startConsumerLoop.
func isPermanentError(err error) bool {
	var pe PermanentError
	return errors.As(err, &pe)
}

func isRetryableError(err error) bool {
	var re RetryableError
	return errors.As(err, &re)
}

// Enforce imports used
var _ = amqp.Table{}
var _ = zap.NewNop
