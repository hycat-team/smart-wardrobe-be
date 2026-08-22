package worker

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/config"
	usecase_interfaces "smart-wardrobe-be/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/internal/pkg/logger"
	"smart-wardrobe-be/internal/shared/observability/workerlog"
	shared_worker "smart-wardrobe-be/internal/shared/presentation/worker"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

type PaymentReconciliationWorker struct {
	*shared_worker.BaseWorker
	useCase usecase_interfaces.IPaymentReconciliationUseCase
	cron    *cron.Cron
	log     logger.Interface
	cfg     *config.Config
}

func NewPaymentReconciliationWorker(
	useCase usecase_interfaces.IPaymentReconciliationUseCase,
	log logger.Interface,
	cfg *config.Config,
) *PaymentReconciliationWorker {
	return &PaymentReconciliationWorker{
		BaseWorker: shared_worker.NewBaseWorker("payment_reconciliation"),
		useCase:    useCase,
		cron:       cron.New(cron.WithSeconds()),
		log:        log,
		cfg:        cfg,
	}
}

func (w *PaymentReconciliationWorker) Start(ctx context.Context) {
	w.BaseWorker.Start(ctx)

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
	w.BaseWorker.Stop()
}

func (w *PaymentReconciliationWorker) run() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	run := workerlog.New("payment_reconciliation", workerlog.TriggerCron)
	if err := w.useCase.Reconcile(ctx, run); err != nil {
		run.LogFailure(w.log, err)
		return
	}
	run.LogSuccessIfData(w.log)
}
