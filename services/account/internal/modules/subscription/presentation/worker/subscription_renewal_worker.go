package worker

import (
	"context"
	usecase_interfaces "smart-wardrobe-be/services/account/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/shared/observability/workerlog"
	"smart-wardrobe-be/shared/pkg/logger"
	shared_worker "smart-wardrobe-be/shared/presentation/worker"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

type SubscriptionRenewalWorker struct {
	*shared_worker.BaseWorker
	subUseCase usecase_interfaces.ISubscriptionUseCase
	cronEngine *cron.Cron
	log        logger.Interface
}

func NewSubscriptionRenewalWorker(
	subUseCase usecase_interfaces.ISubscriptionUseCase,
	log logger.Interface,
) *SubscriptionRenewalWorker {
	return &SubscriptionRenewalWorker{
		BaseWorker: shared_worker.NewBaseWorker("subscription_renewal"),
		subUseCase: subUseCase,
		cronEngine: cron.New(cron.WithSeconds()),
		log:        log,
	}
}

func (w *SubscriptionRenewalWorker) Start(ctx context.Context) {
	w.BaseWorker.Start(ctx)

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
	w.BaseWorker.Stop()
}

func (w *SubscriptionRenewalWorker) executeRenewalJob(triggerType string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	if err := w.subUseCase.ProcessUpcomingExpiryNotifications(ctx); err != nil {
		w.log.Warn("Failed to process upcoming expiry notifications", zap.Error(err))
	}

	run := workerlog.New("subscription_renewal", triggerType)
	if err := w.subUseCase.ProcessScheduledRenewals(ctx, run); err != nil {
		run.LogFailure(w.log, err)
		return
	}
	run.LogSuccessIfData(w.log)
}
