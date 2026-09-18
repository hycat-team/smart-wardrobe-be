package worker

import (
	"context"
	"time"

	uc_interfaces "smart-wardrobe-be/services/commerce/internal/modules/fashion/application/interface/usecase"
	"smart-wardrobe-be/shared/observability/workerlog"
	"smart-wardrobe-be/shared/pkg/logger"
	shared_worker "smart-wardrobe-be/shared/presentation/worker"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

type FailedItemsCleanupWorker struct {
	*shared_worker.BaseWorker
	useCase    uc_interfaces.IFashionWorkerUseCase
	cronEngine *cron.Cron
	log        logger.Interface
}

func NewFailedItemsCleanupWorker(
	useCase uc_interfaces.IFashionWorkerUseCase,
	log logger.Interface,
) *FailedItemsCleanupWorker {
	return &FailedItemsCleanupWorker{
		BaseWorker: shared_worker.NewBaseWorker("failed_items_cleanup"),
		useCase:    useCase,
		cronEngine: cron.New(cron.WithSeconds()),
		log:        log,
	}
}

func (w *FailedItemsCleanupWorker) Start(ctx context.Context) {
	w.BaseWorker.Start(ctx)

	go w.executeCleanupJob(workerlog.TriggerStartup)

	_, err := w.cronEngine.AddFunc("0 0 2 * * *", func() {
		w.executeCleanupJob(workerlog.TriggerCron)
	})
	if err != nil {
		w.log.Error("Failed to register failed items cleanup cron task", zap.Error(err))
		return
	}

	w.cronEngine.Start()
}

func (w *FailedItemsCleanupWorker) Stop() {
	if w.cronEngine != nil {
		w.cronEngine.Stop()
	}
	w.BaseWorker.Stop()
}

func (w *FailedItemsCleanupWorker) executeCleanupJob(triggerType string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	run := workerlog.New("failed_items_cleanup", triggerType)
	if err := w.useCase.CleanupFailedItems(ctx, run); err != nil {
		run.LogFailure(w.log, err)
		return
	}
	run.LogSuccessIfData(w.log)
}
