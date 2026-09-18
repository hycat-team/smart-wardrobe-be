package worker

import (
	"context"
	"time"

	"smart-wardrobe-be/shared/application/interface/usecase"
	"smart-wardrobe-be/shared/config"
	"smart-wardrobe-be/shared/observability/workerlog"
	"smart-wardrobe-be/shared/pkg/logger"

	"go.uber.org/zap"
)

const outboxFallbackInterval = 2 * time.Second
const outboxGaugeInterval = 30 * time.Second

// OutboxSignalListener is the consumer-side port for the infrastructure
// LISTEN/NOTIFY wake-up signal; the concrete listener is provided by DI
// (feature 017 removes the direct infrastructure dependency from here).
type OutboxSignalListener interface {
	SignalChan() <-chan struct{}
}

type OutboxWorker struct {
	*BaseWorker
	outboxUC usecase.IOutboxUseCase
	listener OutboxSignalListener
	logger   logger.Interface
	cfg      config.OutboxConfig
}

func NewOutboxWorker(
	outboxUC usecase.IOutboxUseCase,
	listener OutboxSignalListener,
	cfg *config.Config,
	l logger.Interface,
) *OutboxWorker {
	return &OutboxWorker{
		BaseWorker: NewBaseWorker("outbox"),
		outboxUC:   outboxUC,
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

	gaugeTicker := time.NewTicker(outboxGaugeInterval)
	defer gaugeTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("[OutboxWorker] Context cancelled, stopping")
			return
		case <-w.listener.SignalChan():
			w.processBatch(ctx)
		case <-pollTicker.C:
			w.processBatch(ctx)
		case <-gaugeTicker.C:
			w.refreshGauges(ctx)
		case <-cleanupTicker.C:
			w.cleanupPublished(ctx)
		}
	}
}

// refreshGauges cập nhật gauges outbox_pending/queue_depth/dlq_depth.
func (w *OutboxWorker) refreshGauges(ctx context.Context) {
	w.outboxUC.RefreshGauges(ctx)
}

func (w *OutboxWorker) processBatch(ctx context.Context) {
	retryAfter := time.Duration(w.cfg.RetryIntervalMinutes) * time.Minute

	run := workerlog.New("outbox", workerlog.TriggerCron)
	publishedCount, err := w.outboxUC.PublishBatch(ctx, w.cfg.BatchSize, w.cfg.RetryMaxAttempts, retryAfter)
	if err != nil {
		run.LogFailure(w.logger, err)
		return
	}

	run.AddTotal(publishedCount)
	run.AddSuccess(publishedCount)
	run.LogSuccessIfData(w.logger)
}

func (w *OutboxWorker) cleanupPublished(ctx context.Context) {
	run := workerlog.New("outbox_cleanup", workerlog.TriggerCron)
	deleted, err := w.outboxUC.CleanupPublished(ctx, w.cfg.CleanupRetentionDays)
	if err != nil {
		run.LogFailure(w.logger, err)
		return
	}
	if deleted > 0 {
		run.AddSuccess(int(deleted))
		run.LogSuccessIfData(w.logger, zap.Int64("deleted_rows", deleted))
	}
}

func (w *OutboxWorker) Stop() {
	w.BaseWorker.Stop()
}
