package worker

import (
	"context"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/modules/subscription/contract"
	"smart-wardrobe-be/pkg/logger"
	"time"
)

type IAIUsageReconciliationWorker interface {
	Start()
	Stop()
}
type AIUsageReconciliationWorker struct {
	policy contract.IAICostPolicyContract
	cron   *cron.Cron
	log    logger.Interface
	cfg    *config.Config
}

func NewAIUsageReconciliationWorker(policy contract.IAICostPolicyContract, log logger.Interface, cfg *config.Config) IAIUsageReconciliationWorker {
	return &AIUsageReconciliationWorker{policy: policy, cron: cron.New(cron.WithSeconds()), log: log, cfg: cfg}
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
	count, err := w.policy.ExpireUnknown(ctx, time.Now(), w.cfg.AI.UsageReconcileBatchSize)
	if err != nil {
		w.log.Error("AI usage reconciliation failed", zap.Error(err))
		return
	}
	if count > 0 {
		w.log.Info("AI usage reservations reconciled", zap.Int64("count", count))
	}
}
