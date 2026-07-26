package worker

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/pkg/logger"
	"smart-wardrobe-be/internal/shared/infrastructure/db"
	smart_msg "smart-wardrobe-be/internal/shared/infrastructure/messaging"
	"smart-wardrobe-be/internal/shared/observability/workerlog"

	"go.uber.org/zap"
)

const outboxBatchSize = 50
const outboxFallbackInterval = 10 * time.Second

type OutboxWorker struct {
	*BaseWorker
	outboxRepo smart_msg.OutboxRepository
	rabbitMQ   *smart_msg.RabbitMQClient
	listener   *db.OutboxListener
	logger     logger.Interface
}

func NewOutboxWorker(
	outboxRepo smart_msg.OutboxRepository,
	rabbitMQ *smart_msg.RabbitMQClient,
	listener *db.OutboxListener,
	l logger.Interface,
) *OutboxWorker {
	return &OutboxWorker{
		BaseWorker: NewBaseWorker("outbox"),
		outboxRepo: outboxRepo,
		rabbitMQ:   rabbitMQ,
		listener:   listener,
		logger:     l,
	}
}

func (w *OutboxWorker) Start(ctx context.Context) {
	w.BaseWorker.Start(ctx)
	go w.runLoop(ctx)
}

func (w *OutboxWorker) runLoop(ctx context.Context) {
	pollTicker := time.NewTicker(outboxFallbackInterval)
	defer pollTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("[OutboxWorker] Context cancelled, stopping")
			return
		case <-w.listener.SignalChan():
			w.processBatch(ctx)
		case <-pollTicker.C:
			w.processBatch(ctx)
		}
	}
}

func (w *OutboxWorker) processBatch(ctx context.Context) {
	records, err := w.outboxRepo.ClaimPending(ctx, outboxBatchSize)
	if err != nil {
		w.logger.Error("[OutboxWorker] Failed to claim pending records", zap.Error(err))
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
