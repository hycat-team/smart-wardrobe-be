package worker

import (
	"context"
	uc_interfaces "smart-wardrobe-be/internal/modules/fashion/application/interface/usecase"
	"smart-wardrobe-be/internal/shared/observability/workerlog"
	"smart-wardrobe-be/pkg/logger"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

type IFailedItemsCleanupWorker interface {
	Start()
	Stop()
}

type FailedItemsCleanupWorker struct {
	useCase    uc_interfaces.IFashionWorkerUseCase
	cronEngine *cron.Cron
	log        logger.Interface
}

func NewFailedItemsCleanupWorker(
	useCase uc_interfaces.IFashionWorkerUseCase,
	log logger.Interface,
) IFailedItemsCleanupWorker {
	return &FailedItemsCleanupWorker{
		useCase:    useCase,
		cronEngine: cron.New(cron.WithSeconds()),
		log:        log,
	}
}

func (w *FailedItemsCleanupWorker) Start() {
	go w.executeCleanupJob(workerlog.TriggerStartup)

	_, err := w.cronEngine.AddFunc("0 0 2 * * *", func() {
		w.executeCleanupJob(workerlog.TriggerCron)
	})
	if err != nil {
		w.log.Error("Failed to register scheduled failed items cleanup cron task", zap.Error(err))
		return
	}

	w.cronEngine.Start()
}

func (w *FailedItemsCleanupWorker) Stop() {
	if w.cronEngine != nil {
		w.cronEngine.Stop()
	}
}

func (w *FailedItemsCleanupWorker) executeCleanupJob(triggerType string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	run := workerlog.New("failed_items_cleanup", triggerType)
	if err := w.useCase.CleanupFailedItems(ctx, run); err != nil {
		run.LogFailure(w.log, err)
		return
	}
	run.LogSuccess(w.log)
}
