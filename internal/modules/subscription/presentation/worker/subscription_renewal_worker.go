package worker

import (
	"context"
	usecase_interfaces "smart-wardrobe-be/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/internal/shared/observability/workerlog"
	"smart-wardrobe-be/pkg/logger"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

type ISubscriptionRenewalWorker interface {
	Start()
	Stop()
}

type SubscriptionRenewalWorker struct {
	subUseCase usecase_interfaces.ISubscriptionUseCase
	cronEngine *cron.Cron
	log        logger.Interface
}

func NewSubscriptionRenewalWorker(
	subUseCase usecase_interfaces.ISubscriptionUseCase,
	log logger.Interface,
) ISubscriptionRenewalWorker {
	return &SubscriptionRenewalWorker{
		subUseCase: subUseCase,
		cronEngine: cron.New(cron.WithSeconds()),
		log:        log,
	}
}

func (w *SubscriptionRenewalWorker) Start() {
	go w.executeRenewalJob(workerlog.TriggerStartup)

	_, err := w.cronEngine.AddFunc("0 0 0 * * *", func() {
		w.executeRenewalJob(workerlog.TriggerCron)
	})
	if err != nil {
		w.log.Error("Failed to register scheduled auto-renewal cron task", zap.Error(err))
		return
	}

	w.cronEngine.Start()
}

func (w *SubscriptionRenewalWorker) Stop() {
	if w.cronEngine != nil {
		w.cronEngine.Stop()
	}
}

func (w *SubscriptionRenewalWorker) executeRenewalJob(triggerType string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	run := workerlog.New("subscription_renewal", triggerType)
	if err := w.subUseCase.ProcessScheduledRenewals(ctx, run); err != nil {
		run.LogFailure(w.log, err)
		return
	}
	run.LogSuccess(w.log)
}
