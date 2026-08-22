package worker

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/config"
	"smart-wardrobe-be/internal/modules/aiexecutor/application/interface/usecase"
	"smart-wardrobe-be/internal/pkg/logger"
	"smart-wardrobe-be/internal/shared/observability/workerlog"
	shared_worker "smart-wardrobe-be/internal/shared/presentation/worker"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

type AIUsageReconciliationWorker struct {
	*shared_worker.BaseWorker
	costUC usecase.IAICostUseCase
	cron   *cron.Cron
	log    logger.Interface
	cfg    *config.Config
}

func NewAIUsageReconciliationWorker(costUC usecase.IAICostUseCase, log logger.Interface, cfg *config.Config) *AIUsageReconciliationWorker {
	return &AIUsageReconciliationWorker{
		BaseWorker: shared_worker.NewBaseWorker("ai_usage_reconciliation"),
		costUC:     costUC,
		cron:       cron.New(cron.WithSeconds()),
		log:        log,
		cfg:        cfg,
	}
}

func (w *AIUsageReconciliationWorker) Start(ctx context.Context) {
	w.BaseWorker.Start(ctx)

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
	w.BaseWorker.Stop()
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
	run.LogSuccessIfData(w.log, zap.Int64("expiredReservationCount", count))
}
