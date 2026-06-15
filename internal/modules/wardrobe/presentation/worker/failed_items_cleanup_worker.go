package worker

import (
	"context"
	uc_interfaces "smart-wardrobe-be/internal/modules/wardrobe/application/interface/usecase"
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
	useCase    uc_interfaces.IWardrobeWorkerUseCase
	cronEngine *cron.Cron
	log        logger.Interface
}

func NewFailedItemsCleanupWorker(
	useCase uc_interfaces.IWardrobeWorkerUseCase,
	log logger.Interface,
) IFailedItemsCleanupWorker {
	return &FailedItemsCleanupWorker{
		useCase:    useCase,
		cronEngine: cron.New(cron.WithSeconds()),
		log:        log,
	}
}

func (w *FailedItemsCleanupWorker) Start() {
	go w.executeCleanupJob()

	_, err := w.cronEngine.AddFunc("0 0 2 * * *", w.executeCleanupJob)
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

func (w *FailedItemsCleanupWorker) executeCleanupJob() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	if err := w.useCase.CleanupFailedItems(ctx); err != nil {
		w.log.Error("[FailedItemsCleanupWorker] Job failed", zap.Error(err))
	} else {
		w.log.Info("[FailedItemsCleanupWorker] Job succeeded")
	}
}
