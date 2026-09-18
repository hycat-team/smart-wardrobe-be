package worker

import (
	"context"
	"time"

	uc_interfaces "smart-wardrobe-be/services/commerce/internal/modules/brand/application/interface/usecase"
	"smart-wardrobe-be/shared/config"
	shareddb "smart-wardrobe-be/shared/infrastructure/db"
	"smart-wardrobe-be/shared/observability/workerlog"
	"smart-wardrobe-be/shared/pkg/logger"
	shared_worker "smart-wardrobe-be/shared/presentation/worker"

	"go.uber.org/zap"
)

type LoyaltyPointExpiryWorker struct {
	*shared_worker.BaseWorker
	useCase uc_interfaces.IBrandLoyaltyUseCase
	locker  *shareddb.JobLocker
	log     logger.Interface
	cfg     *config.Config
	stopCh  chan struct{}
}

func NewLoyaltyPointExpiryWorker(
	useCase uc_interfaces.IBrandLoyaltyUseCase,
	locker *shareddb.JobLocker,
	log logger.Interface,
	cfg *config.Config,
) *LoyaltyPointExpiryWorker {
	return &LoyaltyPointExpiryWorker{
		BaseWorker: shared_worker.NewBaseWorker("loyalty_point_expiry"),
		useCase:    useCase,
		locker:     locker,
		log:        log,
		cfg:        cfg,
		stopCh:     make(chan struct{}),
	}
}

func (w *LoyaltyPointExpiryWorker) Start(ctx context.Context) {
	w.BaseWorker.Start(ctx)

	if !w.cfg.Loyalty.ExpiryWorkerEnabled {
		w.log.Info("Loyalty point expiry worker disabled")
		return
	}
	go w.loop()
}

func (w *LoyaltyPointExpiryWorker) Stop() {
	select {
	case <-w.stopCh:
	default:
		close(w.stopCh)
	}
	w.BaseWorker.Stop()
}

func (w *LoyaltyPointExpiryWorker) loop() {
	w.run(workerlog.TriggerStartup)

	ticker := time.NewTicker(w.cfg.Loyalty.ExpiryWorkerInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.run(workerlog.TriggerCron)
		case <-w.stopCh:
			return
		}
	}
}

func (w *LoyaltyPointExpiryWorker) run(triggerType string) {
	ctx, cancel := context.WithTimeout(context.Background(), w.cfg.Loyalty.ExpiryWorkerInterval)
	defer cancel()

	// FR-045/SC-019: expiry is not idempotent — only the instance holding the
	// advisory lock executes it. A skipped run is picked up by the next tick.
	release, acquired, err := w.locker.TryLock(ctx, shareddb.LoyaltyExpiryLockID)
	if err != nil {
		w.log.Warn("Failed to acquire loyalty expiry lock; skipping run", zap.Error(err))
		return
	}
	if !acquired {
		w.log.Info("Loyalty expiry already running on another instance; skipping run")
		return
	}
	defer release()

	run := workerlog.New("loyalty_point_expiry", triggerType)
	expiredPoints, err := w.useCase.ProcessExpiredLoyaltyPointLots(ctx, run, time.Now().UTC(), w.cfg.Loyalty.ExpiryWorkerBatchSize)
	if err != nil {
		run.LogFailure(w.log, err)
		return
	}
	run.LogSuccessIfData(w.log, zap.Int("expiredPoints", expiredPoints))
}
