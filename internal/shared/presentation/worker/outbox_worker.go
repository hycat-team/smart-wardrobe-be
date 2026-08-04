package worker

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/config"
	"smart-wardrobe-be/internal/pkg/logger"
	"smart-wardrobe-be/internal/shared/infrastructure/db"
	smart_msg "smart-wardrobe-be/internal/shared/infrastructure/messaging"
	"smart-wardrobe-be/internal/shared/observability/workerlog"

	"go.uber.org/zap"
)

const outboxFallbackInterval = 10 * time.Second

type OutboxWorker struct {
	*BaseWorker
	outboxRepo smart_msg.OutboxRepository
	rabbitMQ   *smart_msg.RabbitMQClient
	listener   *db.OutboxListener
	logger     logger.Interface
	cfg        config.OutboxConfig
}

func NewOutboxWorker(
	outboxRepo smart_msg.OutboxRepository,
	rabbitMQ *smart_msg.RabbitMQClient,
	listener *db.OutboxListener,
	cfg *config.Config,
	l logger.Interface,
) *OutboxWorker {
	return &OutboxWorker{
		BaseWorker: NewBaseWorker("outbox"),
		outboxRepo: outboxRepo,
		rabbitMQ:   rabbitMQ,
		listener:   listener,
		logger:     l,
		cfg:        cfg.Outbox,
	}
}

func (w *OutboxWorker) Start(ctx context.Context) {
	w.BaseWorker.Start(ctx)
	go w.runLoop(ctx)
}

func (w *OutboxWorker) runLoop(ctx context.Context) {
	pollTicker := time.NewTicker(outboxFallbackInterval)
	defer pollTicker.Stop()

	cleanupInterval := time.Duration(w.cfg.CleanupIntervalMinutes) * time.Minute
	if cleanupInterval <= 0 {
		cleanupInterval = 1 * time.Hour
	}
	cleanupTicker := time.NewTicker(cleanupInterval)
	defer cleanupTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("[OutboxWorker] Context cancelled, stopping")
			return
		case <-w.listener.SignalChan():
			w.processBatch(ctx)
		case <-pollTicker.C:
			w.processBatch(ctx)
		case <-cleanupTicker.C:
			w.cleanupPublished(ctx)
		}
	}
}

func (w *OutboxWorker) processBatch(ctx context.Context) {
	retryAfter := time.Duration(w.cfg.RetryIntervalMinutes) * time.Minute
	records, err := w.outboxRepo.ClaimToPublish(ctx, w.cfg.BatchSize, w.cfg.RetryMaxAttempts, retryAfter)
	if err != nil {
		w.logger.Error("[OutboxWorker] Failed to claim records ready to publish", zap.Error(err))
		return
	}
	if len(records) == 0 {
		return
	}

	run := workerlog.New("outbox", workerlog.TriggerCron)
	run.AddTotal(len(records))

	for _, record := range records {
		err := w.rabbitMQ.PublishDirect(ctx, record.Topic, record.Payload)
		if err != nil {
			w.logger.Error("[OutboxWorker] Failed to publish outbox record",
				zap.String("recordId", record.ID.String()),
				zap.String("topic", record.Topic),
				zap.Error(err),
			)
			if markErr := w.outboxRepo.MarkFailed(ctx, record.ID, err); markErr != nil {
				w.logger.Error("[OutboxWorker] Failed to mark outbox record as failed",
					zap.String("recordId", record.ID.String()),
					zap.Error(markErr),
				)
			}
			continue
		}

		if err := w.outboxRepo.MarkPublished(ctx, record.ID); err != nil {
			w.logger.Error("[OutboxWorker] Failed to mark outbox record as published",
				zap.String("recordId", record.ID.String()),
				zap.Error(err),
			)
		}
		run.AddSuccess(1)
	}

	run.LogSuccessIfData(w.logger)
}

func (w *OutboxWorker) cleanupPublished(ctx context.Context) {
	retention := time.Duration(w.cfg.CleanupRetentionDays) * 24 * time.Hour
	since := time.Now().UTC().Add(-retention)
	deleted, err := w.outboxRepo.DeletePublishedBefore(ctx, since)
	if err != nil {
		w.logger.Error("[OutboxWorker] Failed to clean up old published records",
			zap.Time("since", since),
			zap.Error(err),
		)
		return
	}
	if deleted > 0 {
		w.logger.Info("[OutboxWorker] Cleaned up old published records",
			zap.Int64("count", deleted),
			zap.Time("since", since),
		)
	}
}
