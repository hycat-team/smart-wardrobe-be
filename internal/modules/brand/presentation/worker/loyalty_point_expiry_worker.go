package worker

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/config"
	uc_interfaces "smart-wardrobe-be/internal/modules/brand/application/interface/usecase"
	"smart-wardrobe-be/internal/pkg/logger"
	"smart-wardrobe-be/internal/shared/observability/workerlog"
	shared_worker "smart-wardrobe-be/internal/shared/presentation/worker"

	"go.uber.org/zap"
)

type LoyaltyPointExpiryWorker struct {
	*shared_worker.BaseWorker
	useCase uc_interfaces.IBrandLoyaltyUseCase
	log     logger.Interface
	cfg     *config.Config
	stopCh  chan struct{}
}

func NewLoyaltyPointExpiryWorker(
	useCase uc_interfaces.IBrandLoyaltyUseCase,
	log logger.Interface,
	cfg *config.Config,
) *LoyaltyPointExpiryWorker {
	return &LoyaltyPointExpiryWorker{
		BaseWorker: shared_worker.NewBaseWorker("loyalty_point_expiry"),
		useCase:    useCase,
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

	run := workerlog.New("loyalty_point_expiry", triggerType)
	expiredPoints, err := w.useCase.ProcessExpiredLoyaltyPointLots(ctx, run, time.Now().UTC(), w.cfg.Loyalty.ExpiryWorkerBatchSize)
	if err != nil {
		run.LogFailure(w.log, err)
		return
	}
	run.LogSuccessIfData(w.log, zap.Int("expiredPoints", expiredPoints))
}
