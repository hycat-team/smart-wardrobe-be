package messaging

import (
	"context"
	"errors"

	"smart-wardrobe-be/shared/application/event"

	"go.uber.org/zap"
)

// RegisterTypedHandler đăng ký 1 handler nhận (ctx, event.Message) cho queue:
// resolve Prefetch/Concurrency defaults từ Topology registry (single source
// of truth cho cấu hình per-queue), inject correlationId vào context
// per-message, rồi spawn Concurrency consumer goroutines qua RegisterConsumer
// (registry này cũng được reRegisterConsumers tái sử dụng khi reconnect).
// Handler trả event.PermanentError → DLQ, event.RetryableError → requeue
// (quyết định ack/nack nằm ở startConsumerLoop).
func (r *RabbitMQClient) RegisterTypedHandler(queue string, handler func(ctx context.Context, msg event.Message) error, opts event.ConsumerOpts) error {
	if opts.Concurrency <= 0 {
		opts.Concurrency = topologyConcurrency(queue)
	}
	if opts.Prefetch <= 0 {
		opts.Prefetch = topologyPrefetch(queue)
	}

	wrapped := func(msg event.Message) error {
		ctx := event.ContextWithCorrelationId(r.baseCtx, msg)
		if err := handler(ctx, msg); err != nil {
			var pe event.PermanentError
			var re event.RetryableError
			logFields := []zap.Field{
				zap.String("queue", queue),
				zap.String("correlationId", msg.CorrelationId()),
				zap.Error(err),
			}
			// 024 T152: this used to have two branches - "poison message" for
			// PermanentError and "will retry" for everything else. But
			// `startConsumerLoop` has three: it requeues ONLY a RetryableError and
			// dead-letters the default. So a handler that returned an unclassified
			// error was logged as retrying while being discarded on the spot, and
			// the log - the only record of what happened to a message - was wrong.
			//
			// The three branches now mirror the three outcomes exactly.
			switch {
			case errors.As(err, &re):
				r.logger.Error("[TypedHandler] Transient handler error, message will be requeued", logFields...)
			case errors.As(err, &pe):
				r.logger.Error("[TypedHandler] Poison message, dead-lettering", logFields...)
			default:
				r.logger.Error("[TypedHandler] Unclassified handler error, dead-lettering; "+
					"wrap it in event.RetryableError if redelivery could help", logFields...)
			}
			return err
		}
		return nil
	}

	for i := 0; i < opts.Concurrency; i++ {
		r.RegisterConsumer(queue, wrapped)
	}
	r.logger.Info("[TypedHandler] Registered typed handler",
		zap.String("queue", queue),
		zap.Int("concurrency", opts.Concurrency),
		zap.Int("prefetch", opts.Prefetch),
	)
	return nil
}

// topologyConcurrency/topologyPrefetch đọc cấu hình per-queue từ registry
// Topology — single source of truth cho cấu hình queue, fallback 1/20.
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
