package worker

import (
	"context"
	"time"

	"smart-wardrobe-be/config"
	usecase_interfaces "smart-wardrobe-be/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/internal/shared/observability/workerlog"
	"smart-wardrobe-be/pkg/logger"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

type IPaymentReconciliationWorker interface {
	Start()
	Stop()
}

type PaymentReconciliationWorker struct {
	useCase usecase_interfaces.IPaymentReconciliationUseCase
	cron    *cron.Cron
	log     logger.Interface
	cfg     *config.Config
}

func NewPaymentReconciliationWorker(
	useCase usecase_interfaces.IPaymentReconciliationUseCase,
	log logger.Interface,
	cfg *config.Config,
) IPaymentReconciliationWorker {
	return &PaymentReconciliationWorker{
		useCase: useCase,
		cron:    cron.New(cron.WithSeconds()),
		log:     log,
		cfg:     cfg,
	}
}

func (w *PaymentReconciliationWorker) Start() {
	_, err := w.cron.AddFunc(w.cfg.PayOS.ReconciliationCron, w.run)
	if err != nil {
		w.log.Error("Failed to register payment reconciliation worker", zap.Error(err))
		return
	}
	w.cron.Start()
}

func (w *PaymentReconciliationWorker) Stop() {
	if w.cron != nil {
		w.cron.Stop()
	}
}

func (w *PaymentReconciliationWorker) run() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	run := workerlog.New("payment_reconciliation", workerlog.TriggerCron)
	if err := w.useCase.Reconcile(ctx, run); err != nil {
		run.LogFailure(w.log, err)
		return
	}
	run.LogSuccess(w.log)
}
