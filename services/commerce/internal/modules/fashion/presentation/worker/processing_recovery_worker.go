package worker

import (
	"context"
	"time"

	uc_interfaces "smart-wardrobe-be/services/commerce/internal/modules/fashion/application/interface/usecase"
	"smart-wardrobe-be/shared/config"
	"smart-wardrobe-be/shared/observability/workerlog"
	"smart-wardrobe-be/shared/pkg/logger"
	shared_worker "smart-wardrobe-be/shared/presentation/worker"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

type ProcessingRecoveryWorker struct {
	*shared_worker.BaseWorker
	useCase    uc_interfaces.IFashionWorkerUseCase
	cronEngine *cron.Cron
	log        logger.Interface
	cfg        *config.Config
}

func NewProcessingRecoveryWorker(
	cfg *config.Config,
	useCase uc_interfaces.IFashionWorkerUseCase,
	log logger.Interface,
) *ProcessingRecoveryWorker {
	return &ProcessingRecoveryWorker{
		BaseWorker: shared_worker.NewBaseWorker("processing_recovery"),
		useCase:    useCase,
		cronEngine: cron.New(cron.WithSeconds()),
		log:        log,
		cfg:        cfg,
	}
}

func (w *ProcessingRecoveryWorker) Start(ctx context.Context) {
	w.BaseWorker.Start(ctx)

	go w.executeRecoveryJob(workerlog.TriggerStartup)

	_, err := w.cronEngine.AddFunc(w.cfg.Wardrobe.RecoveryScanCron, func() {
		w.executeRecoveryJob(workerlog.TriggerCron)
	})
	if err != nil {
		w.log.Error("Failed to register wardrobe processing recovery cron task", zap.Error(err))
		return
	}

	w.cronEngine.Start()
}

func (w *ProcessingRecoveryWorker) Stop() {
	if w.cronEngine != nil {
		w.cronEngine.Stop()
	}
	w.BaseWorker.Stop()
}

func (w *ProcessingRecoveryWorker) executeRecoveryJob(triggerType string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	run := workerlog.New("processing_recovery", triggerType)
	if err := w.useCase.RecoverStaleProcessingItems(ctx, run); err != nil {
		run.LogFailure(w.log, err)
		return
	}
	run.LogSuccessIfData(w.log)
}
