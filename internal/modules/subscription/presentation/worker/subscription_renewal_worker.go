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
	// Execute the renewal validation check immediately on application startup in a separate goroutine
	go w.executeRenewalJob()

	// Register the scheduled daily renewal check to run periodically every midnight
	_, err := w.cronEngine.AddFunc("0 0 0 * * *", func() {
		w.log.Info("Daily subscription renewal process tick triggered at midnight")
		w.executeRenewalJob()
	})

	if err != nil {
		w.log.Error("Failed to register scheduled auto-renewal cron task", zap.Error(err))
		return
	}

	w.cronEngine.Start()
	w.log.Info("Subscription auto-renewal background scheduler started successfully")
}

func (w *SubscriptionRenewalWorker) Stop() {
	if w.cronEngine != nil {
		w.cronEngine.Stop()
		w.log.Info("Subscription auto-renewal background scheduler stopped safely")
	}
}

func (w *SubscriptionRenewalWorker) executeRenewalJob() {
	w.log.Info("Initiating background subscription renewal execution cycle")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	if err := w.subUseCase.ProcessScheduledRenewals(ctx); err != nil {
		w.log.Error("Subscription renewal scheduled process execution failure", zap.Error(err))
	} else {
		w.log.Info("Subscription renewal scheduled process execution completed successfully")
	}
}
