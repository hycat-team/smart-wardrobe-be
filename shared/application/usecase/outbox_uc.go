package usecase

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"smart-wardrobe-be/shared/application/event"
	"smart-wardrobe-be/shared/config"
	shared_repos "smart-wardrobe-be/shared/domain/repositories"
	"smart-wardrobe-be/shared/pkg/logger"

	"go.uber.org/zap"
)

type OutboxUseCase struct {
	outboxRepo shared_repos.OutboxRepository
	publisher  event.IRawMessagePublisher
	gaugeProbe event.IOutboxGaugeProbe
	cfg        *config.Config
	logger     logger.Interface
}

func NewOutboxUseCase(
	outboxRepo shared_repos.OutboxRepository,
	publisher event.IRawMessagePublisher,
	gaugeProbe event.IOutboxGaugeProbe,
	cfg *config.Config,
	l logger.Interface,
) *OutboxUseCase {
	return &OutboxUseCase{
		outboxRepo: outboxRepo,
		publisher:  publisher,
		gaugeProbe: gaugeProbe,
		cfg:        cfg,
		logger:     l,
	}
}

// RefreshGauges cập nhật gauge outbox_pending từ DB; queue_depth/dlq_depth
// được probe infra-side. Lỗi chỉ log warn — metrics không được phép giết worker.
func (uc *OutboxUseCase) RefreshGauges(ctx context.Context) {
	pending, err := uc.outboxRepo.CountPending(ctx)
	if err != nil {
		uc.logger.Warn("[OutboxUseCase] Failed to count pending outbox records", zap.Error(err))
		return
	}
	uc.gaugeProbe.ProbeGauges(ctx, pending)
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

	// Parallel publish with bounded concurrency
	type result struct {
		id         string
		published  bool
		publishErr error
	}
	results := make([]result, len(records))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 10)

	for i, record := range records {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			headers := map[string]any{}
			if len(record.Headers) > 0 && string(record.Headers) != "{}" {
				var h map[string]any
				if err := json.Unmarshal(record.Headers, &h); err == nil {
					for k, v := range h {
						if k == "correlationId" {
							headers["x-correlation-id"] = v
						}
						headers[k] = v
					}
				}
			}
			if err := uc.publisher.PublishDirect(ctx, record.Topic, record.Payload, headers); err != nil {
				results[i] = result{id: record.ID.String(), publishErr: err}
				return
			}
			results[i] = result{id: record.ID.String(), published: true}
		}()
	}
	wg.Wait()

	publishedCount := 0
	for i, record := range records {
		r := results[i]
		if r.publishErr != nil {
			uc.logger.Error("[OutboxUseCase] Failed to publish outbox record",
				zap.String("recordId", record.ID.String()),
				zap.String("topic", record.Topic),
				zap.Error(r.publishErr),
			)
			if markErr := uc.outboxRepo.MarkFailed(ctx, record.ID, r.publishErr); markErr != nil {
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
		if r.published {
			publishedCount++
		}
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
