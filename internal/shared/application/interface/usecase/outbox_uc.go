package usecase

import (
	"context"
	"time"
)

type IOutboxUseCase interface {
	PublishBatch(ctx context.Context, batchSize int, retryMax int, retryAfter time.Duration) (int, error)
	CleanupPublished(ctx context.Context, retentionDays int) (int64, error)
	// RefreshGauges cập nhật gauges outbox_pending/queue_depth/dlq_depth.
	RefreshGauges(ctx context.Context)
}
