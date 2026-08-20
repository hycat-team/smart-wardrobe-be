package worker

import (
	"context"
	"time"

	uc_interfaces "smart-wardrobe-be/internal/modules/admin/application/interface/usecase"
	"smart-wardrobe-be/internal/pkg/logger"
	"smart-wardrobe-be/internal/shared/observability/workerlog"
	"smart-wardrobe-be/internal/shared/presentation/worker"

	"go.uber.org/zap"
)

type AuditCleanupWorker struct {
	*worker.BaseWorker
	auditLogUC uc_interfaces.IAuditLogUseCase
	logger     logger.Interface
	ticker     *time.Ticker
}

func NewAuditCleanupWorker(
	auditLogUC uc_interfaces.IAuditLogUseCase,
	l logger.Interface,
) *AuditCleanupWorker {
	return &AuditCleanupWorker{
		BaseWorker: worker.NewBaseWorker("admin_audit_cleanup"),
		auditLogUC: auditLogUC,
		logger:     l,
	}
}

func (w *AuditCleanupWorker) Start(ctx context.Context) {
	w.BaseWorker.Start(ctx)
	// Run cleanup once every 24 hours
	w.ticker = time.NewTicker(24 * time.Hour)

	go func() {
		// Run once on startup
		w.runCleanup(ctx)

		for {
			select {
			case <-ctx.Done():
				return
			case <-w.ticker.C:
				w.runCleanup(ctx)
			}
		}
	}()
}

func (w *AuditCleanupWorker) runCleanup(ctx context.Context) {
	run := workerlog.New("admin_audit_cleanup", workerlog.TriggerCron)
	// Retain logs for 12 months (365 days)
	cutoff := time.Now().AddDate(0, -12, 0)

	deletedCount, err := w.auditLogUC.DeleteOlderThan(ctx, cutoff)
	if err != nil {
		run.LogFailure(w.logger, err, zap.Time("cutoff", cutoff))
		return
	}

	if deletedCount > 0 {
		run.AddSuccess(int(deletedCount))
		run.LogSuccessIfData(w.logger, zap.Int64("deleted_rows", deletedCount))
	}
}

func (w *AuditCleanupWorker) Stop() {
	if w.ticker != nil {
		w.ticker.Stop()
	}
	w.BaseWorker.Stop()
}
