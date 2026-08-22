package worker

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/config"
	uc_interfaces "smart-wardrobe-be/internal/modules/admin/application/interface/usecase"
	"smart-wardrobe-be/internal/pkg/logger"
	"smart-wardrobe-be/internal/shared/observability/workerlog"
	"smart-wardrobe-be/internal/shared/presentation/worker"

	"go.uber.org/zap"
)

type AuditCleanupWorker struct {
	*worker.BaseWorker
	auditLogUC      uc_interfaces.IAuditLogUseCase
	retentionMonths int
	logger          logger.Interface
	ticker          *time.Ticker
}

func NewAuditCleanupWorker(
	auditLogUC uc_interfaces.IAuditLogUseCase,
	cfg *config.Config,
	l logger.Interface,
) *AuditCleanupWorker {
	retentionMonths := cfg.Admin.AuditRetentionMonths
	if retentionMonths <= 0 {
		retentionMonths = 12
	}
	return &AuditCleanupWorker{
		BaseWorker:      worker.NewBaseWorker("admin_audit_cleanup"),
		auditLogUC:      auditLogUC,
		retentionMonths: retentionMonths,
		logger:          l,
	}
}

func (w *AuditCleanupWorker) Start(ctx context.Context) {
	w.BaseWorker.Start(ctx)
	w.ticker = time.NewTicker(24 * time.Hour)

	go func() {
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
	cutoff := time.Now().AddDate(0, -w.retentionMonths, 0)

	deletedCount, err := w.auditLogUC.DeleteOlderThan(ctx, cutoff)
	if err != nil {
		run.LogFailure(w.logger, err, zap.Time("cutoff", cutoff))
		return
	}

	if deletedCount > 0 {
		run.AddTotal(int(deletedCount))
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
