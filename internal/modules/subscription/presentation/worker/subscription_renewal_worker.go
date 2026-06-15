package worker

import (
	"context"
	usecase_interfaces "smart-wardrobe-be/internal/modules/subscription/application/interface/usecase"
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
	go w.executeRenewalJob()

	_, err := w.cronEngine.AddFunc("0 0 0 * * *", w.executeRenewalJob)
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

func (w *SubscriptionRenewalWorker) executeRenewalJob() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	if err := w.subUseCase.ProcessScheduledRenewals(ctx); err != nil {
		w.log.Error("[SubscriptionRenewalWorker] Job failed", zap.Error(err))
	} else {
		w.log.Info("[SubscriptionRenewalWorker] Job succeeded")
	}
}
