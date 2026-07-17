package worker

import (
	"context"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"smart-wardrobe-be/internal/config"
	"smart-wardrobe-be/internal/modules/aiexecutor/application/interface/usecase"
	"smart-wardrobe-be/internal/pkg/logger"
	"smart-wardrobe-be/internal/shared/observability/workerlog"
	"time"
)

type IAIUsageReconciliationWorker interface {
	Start()
	Stop()
}
type AIUsageReconciliationWorker struct {
	costUC usecase.IAICostUseCase
	cron   *cron.Cron
	log    logger.Interface
	cfg    *config.Config
}

func NewAIUsageReconciliationWorker(costUC usecase.IAICostUseCase, log logger.Interface, cfg *config.Config) IAIUsageReconciliationWorker {
	return &AIUsageReconciliationWorker{costUC: costUC, cron: cron.New(cron.WithSeconds()), log: log, cfg: cfg}
}
func (w *AIUsageReconciliationWorker) Start() {
	if _, err := w.cron.AddFunc(w.cfg.AI.UsageReconcileCron, w.run); err != nil {
		w.log.Error("Failed to register AI usage reconciliation worker", zap.Error(err))
		return
	}
	w.cron.Start()
}
func (w *AIUsageReconciliationWorker) Stop() {
	if w.cron != nil {
		w.cron.Stop()
	}
}
func (w *AIUsageReconciliationWorker) run() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	run := workerlog.New("ai_usage_reconciliation", workerlog.TriggerCron)
	count, err := w.costUC.ExpireUnknown(ctx, time.Now(), w.cfg.AI.UsageReconcileBatchSize)
	if err != nil {
		run.LogFailure(w.log, err, zap.Int64("expiredReservationCount", 0))
		return
	}
	run.AddTotal(int(count))
	run.AddSuccess(int(count))
	run.LogSuccess(w.log, zap.Int64("expiredReservationCount", count))
}
