package messaging

import (
	"context"
	"errors"

	"smart-wardrobe-be/internal/shared/application/event"

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
			logFields := []zap.Field{
				zap.String("queue", queue),
				zap.String("correlationId", msg.CorrelationId()),
				zap.Error(err),
			}
			if errors.As(err, &pe) {
				r.logger.Error("[TypedHandler] Poison message, dead-lettering", logFields...)
			} else {
				r.logger.Error("[TypedHandler] Handler error, will retry", logFields...)
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
