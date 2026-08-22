package usecase

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/pkg/logger"
	"smart-wardrobe-be/internal/shared/infrastructure/messaging"

	"go.uber.org/zap"
)

type OutboxUseCase struct {
	outboxRepo messaging.OutboxRepository
	rabbitMQ   *messaging.RabbitMQClient
	logger     logger.Interface
}

func NewOutboxUseCase(
	outboxRepo messaging.OutboxRepository,
	rabbitMQ *messaging.RabbitMQClient,
	l logger.Interface,
) *OutboxUseCase {
	return &OutboxUseCase{
		outboxRepo: outboxRepo,
		rabbitMQ:   rabbitMQ,
		logger:     l,
	}
}

func (uc *OutboxUseCase) PublishBatch(ctx context.Context, batchSize int, retryMax int, retryAfter time.Duration) (int, error) {
	records, err := uc.outboxRepo.ClaimToPublish(ctx, batchSize, retryMax, retryAfter)
	if err != nil {
		uc.logger.Error("[OutboxUseCase] Failed to claim records ready to publish", zap.Error(err))
		return 0, err
	}
	if len(records) == 0 {
		return 0, nil
	}

	publishedCount := 0
	for _, record := range records {
		if err := uc.rabbitMQ.PublishDirect(ctx, record.Topic, record.Payload); err != nil {
			uc.logger.Error("[OutboxUseCase] Failed to publish outbox record",
				zap.String("recordId", record.ID.String()),
				zap.String("topic", record.Topic),
				zap.Error(err),
			)
			if markErr := uc.outboxRepo.MarkFailed(ctx, record.ID, err); markErr != nil {
				uc.logger.Error("[OutboxUseCase] Failed to mark outbox record as failed",
					zap.String("recordId", record.ID.String()),
					zap.Error(markErr),
				)
			}
			continue
		}

		if err := uc.outboxRepo.MarkPublished(ctx, record.ID); err != nil {
			uc.logger.Error("[OutboxUseCase] Failed to mark outbox record as published",
				zap.String("recordId", record.ID.String()),
				zap.Error(err),
			)
		}
		publishedCount++
	}

	return publishedCount, nil
}

func (uc *OutboxUseCase) CleanupPublished(ctx context.Context, retentionDays int) (int64, error) {
	retention := time.Duration(retentionDays) * 24 * time.Hour
	since := time.Now().UTC().Add(-retention)
	deleted, err := uc.outboxRepo.DeletePublishedBefore(ctx, since)
	if err != nil {
		uc.logger.Error("[OutboxUseCase] Failed to clean up old published records",
			zap.Time("since", since),
			zap.Error(err),
		)
		return 0, err
	}
	if deleted > 0 {
		uc.logger.Info("[OutboxUseCase] Cleaned up old published records",
			zap.Int64("count", deleted),
			zap.Time("since", since),
		)
	}
	return deleted, nil
}
