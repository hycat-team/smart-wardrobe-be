package messaging

import (
	"context"
	"encoding/json"

	"smart-wardrobe-be/internal/shared/application/event"

	"go.uber.org/zap"
)

// correlationIDContextKey là key lưu correlationId trong context per-message,
// để handler/workerlog log cùng correlationId với phía publisher (outbox).
type correlationIDContextKey struct{}

// CorrelationIdFromContext đọc correlationId từ ctx mà RegisterTypedConsumer inject.
func CorrelationIdFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(correlationIDContextKey{}).(string); ok {
		return v
	}
	return ""
}

// PermanentError indicates a poison message that should go to DLQ (no requeue).
type PermanentError struct{ Err error }

func (e PermanentError) Error() string { return e.Err.Error() }
func (e PermanentError) Unwrap() error { return e.Err }

// RetryableError indicates a transient failure that should be requeued/retried.
type RetryableError struct{ Err error }

func (e RetryableError) Error() string { return e.Err.Error() }
func (e RetryableError) Unwrap() error { return e.Err }

// ConsumerOpts controls per-queue QoS and concurrency.
// Zero values means "đọc từ Topology theo queue" (FR-008/FR-022);
// x-delivery-limit là queue argument (queueArgs), không phải consumer option.
type ConsumerOpts struct {
	Prefetch    int
	Concurrency int
}

// TypedHandler processes a typed payload.
type TypedHandler[T any] func(ctx context.Context, payload T) error

// RegisterTypedConsumer registers a typed consumer for the given queue.
// It unmarshals JSON into T (unmarshal fail → PermanentError → DLQ),
// injects correlationId từ message headers vào context per-message,
// và spawn opts.Concurrency consumer goroutines (default từ Topology).
func RegisterTypedConsumer[T any](r *RabbitMQClient, queue string, handler TypedHandler[T], opts ConsumerOpts) error {
	if opts.Concurrency <= 0 {
		opts.Concurrency = topologyConcurrency(queue)
	}
	if opts.Prefetch <= 0 {
		opts.Prefetch = topologyPrefetch(queue)
	}

	// Wrapper that unmarshals and propagates correlationId
	wrapper := func(msg event.Message) error {
		var payload T
		if err := json.Unmarshal(msg.Body, &payload); err != nil {
			r.logger.Error("[TypedConsumer] Failed to unmarshal message body",
				zap.String("queue", queue),
				zap.String("correlationId", msg.CorrelationId()),
				zap.Error(err),
			)
			return PermanentError{Err: err}
		}
		ctx := context.WithValue(r.baseCtx, correlationIDContextKey{}, msg.CorrelationId())
		return handler(ctx, payload)
	}

	for i := 0; i < opts.Concurrency; i++ {
		r.RegisterConsumer(queue, wrapper)
	}
	r.logger.Info("[TypedConsumer] Registered typed consumer",
		zap.String("queue", queue),
		zap.Int("concurrency", opts.Concurrency),
		zap.Int("prefetch", opts.Prefetch),
	)
	return nil
}

// topologyConcurrency/topologyPrefetch đọc cấu hình per-queue từ registry
// Topology — single source of truth (FR-008/FR-022), fallback 1/20.
func topologyConcurrency(queue string) int {
	for _, b := range Topology {
		if b.Queue == queue && b.Concurrency > 0 {
			return b.Concurrency
		}
	}
	return 1
}

func topologyPrefetch(queue string) int {
	for _, b := range Topology {
		if b.Queue == queue && b.Prefetch > 0 {
			return b.Prefetch
		}
	}
	return 20
}
