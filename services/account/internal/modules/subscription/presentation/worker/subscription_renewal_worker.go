package worker

import (
	"context"
	usecase_interfaces "smart-wardrobe-be/services/account/internal/modules/subscription/application/interface/usecase"
	shareddb "smart-wardrobe-be/shared/infrastructure/db"
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
	locker     *shareddb.JobLocker
	cronEngine *cron.Cron
	log        logger.Interface
}

func NewSubscriptionRenewalWorker(
	subUseCase usecase_interfaces.ISubscriptionUseCase,
	locker *shareddb.JobLocker,
	log logger.Interface,
) *SubscriptionRenewalWorker {
	return &SubscriptionRenewalWorker{
		BaseWorker: shared_worker.NewBaseWorker("subscription_renewal"),
		subUseCase: subUseCase,
		locker:     locker,
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

	run := workerlog.New("subscription_renewal", triggerType)

	// FR-045/SC-019: renewals are not idempotent — only the instance holding
	// the advisory lock executes them, so overlapping schedules or restarts
	// cannot double-charge. A skipped run is picked up by the next tick.
	release, acquired, err := w.locker.TryLock(ctx, shareddb.SubscriptionRenewalLockID)
	if err != nil {
		w.log.Warn("Failed to acquire subscription renewal lock; skipping run", zap.Error(err))
		run.LogFailure(w.log, err)
		return
	}
	if !acquired {
		w.log.Info("Subscription renewal already running on another instance; skipping run")
		return
	}
	defer release()

	// The reminder runs UNDER the same lock, and on the daily tick only.
	//
	// Both halves are deliberate. It used to run before the lock and on startup
	// as well, so a two-instance deployment sent every customer two copies of the
	// same warning, and three restarts in a day sent three more. A reminder is a
	// side effect that leaves the system, exactly like a charge: one runner.
	// Missing a restart-day reminder costs nothing; the next midnight covers it.
	if triggerType == workerlog.TriggerCron {
		if err := w.subUseCase.ProcessUpcomingExpiryNotifications(ctx); err != nil {
			w.log.Warn("Failed to process upcoming expiry notifications", zap.Error(err))
		}
	}

	if err := w.subUseCase.ProcessScheduledRenewals(ctx, run); err != nil {
		run.LogFailure(w.log, err)
		return
	}
	run.LogSuccessIfData(w.log)
}
