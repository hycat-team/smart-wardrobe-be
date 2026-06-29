package worker

import (
	"context"
	"time"

	"smart-wardrobe-be/config"
	uc_interfaces "smart-wardrobe-be/internal/modules/brand/application/interface/usecase"
	"smart-wardrobe-be/internal/shared/observability/workerlog"
	"smart-wardrobe-be/pkg/logger"

	"go.uber.org/zap"
)

type ILoyaltyPointExpiryWorker interface {
	Start()
	Stop()
}

type LoyaltyPointExpiryWorker struct {
	useCase uc_interfaces.IBrandLoyaltyUseCase
	log     logger.Interface
	cfg     *config.Config
	stopCh  chan struct{}
}

func NewLoyaltyPointExpiryWorker(
	useCase uc_interfaces.IBrandLoyaltyUseCase,
	log logger.Interface,
	cfg *config.Config,
) ILoyaltyPointExpiryWorker {
	return &LoyaltyPointExpiryWorker{
		useCase: useCase,
		log:     log,
		cfg:     cfg,
		stopCh:  make(chan struct{}),
	}
}

func (w *LoyaltyPointExpiryWorker) Start() {
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
	expiredPoints, err := w.useCase.ProcessExpiredLoyaltyPointLots(ctx, time.Now().UTC(), w.cfg.Loyalty.ExpiryWorkerBatchSize)
	if err != nil {
		run.LogFailure(w.log, err)
		return
	}
	run.LogSuccess(w.log, zap.Int("expiredPoints", expiredPoints))
}
