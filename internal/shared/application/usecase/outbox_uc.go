package usecase

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"smart-wardrobe-be/internal/config"
	"smart-wardrobe-be/internal/pkg/logger"
	"smart-wardrobe-be/internal/shared/infrastructure/messaging"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

type OutboxUseCase struct {
	outboxRepo messaging.OutboxRepository
	rabbitMQ   *messaging.RabbitMQClient
	cfg        *config.Config
	logger     logger.Interface
}

func NewOutboxUseCase(
	outboxRepo messaging.OutboxRepository,
	rabbitMQ *messaging.RabbitMQClient,
	cfg *config.Config,
	l logger.Interface,
) *OutboxUseCase {
	return &OutboxUseCase{
		outboxRepo: outboxRepo,
		rabbitMQ:   rabbitMQ,
		cfg:        cfg,
		logger:     l,
	}
}

// RefreshGauges cập nhật gauges vận hành: outbox_pending từ DB,
// queue_depth/dlq_depth từ RabbitMQ Management API. Lỗi chỉ log warn —
// metrics không được phép giết worker.
func (uc *OutboxUseCase) RefreshGauges(ctx context.Context) {
	pending, err := uc.outboxRepo.CountPending(ctx)
	if err != nil {
		uc.logger.Warn("[OutboxUseCase] Failed to count pending outbox records", zap.Error(err))
	}

	mgmtPort := uc.cfg.RabbitMQ.ManagementPort
	if mgmtPort <= 0 {
		mgmtPort = 15672
	}
	var queueDepth, dlqDepth int64
	queueDepth, dlqDepth, err = messaging.FetchQueueDepths(ctx, uc.cfg.RabbitMQ.Host, mgmtPort,
		uc.cfg.RabbitMQ.User, uc.cfg.RabbitMQ.Password, uc.logger)
	if err != nil {
		uc.logger.Warn("[OutboxUseCase] Failed to fetch queue depths from management API", zap.Error(err))
	}

	messaging.SetGauges(pending, queueDepth, dlqDepth)
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
			headers := amqp.Table{}
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
			if err := uc.rabbitMQ.PublishDirect(ctx, record.Topic, record.Payload, headers); err != nil {
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
