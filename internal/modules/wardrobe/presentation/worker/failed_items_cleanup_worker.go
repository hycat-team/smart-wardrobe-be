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

	_, err := w.cronEngine.AddFunc("0 0 2 * * *", func() {
		w.log.Info("Daily failed items cleanup process tick triggered at 2:00 AM")
		w.executeCleanupJob()
	})

	if err != nil {
		w.log.Error("Failed to register scheduled failed items cleanup cron task", zap.Error(err))
		return
	}

	w.cronEngine.Start()
	w.log.Info("Failed items cleanup background scheduler started successfully")
}

func (w *FailedItemsCleanupWorker) Stop() {
	if w.cronEngine != nil {
		w.cronEngine.Stop()
		w.log.Info("Failed items cleanup background scheduler stopped safely")
	}
}

func (w *FailedItemsCleanupWorker) executeCleanupJob() {
	w.log.Info("Initiating background failed items cleanup execution cycle")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	if err := w.useCase.CleanupFailedItems(ctx); err != nil {
		w.log.Error("Failed items cleanup process execution failure", zap.Error(err))
	} else {
		w.log.Info("Failed items cleanup process execution completed successfully")
	}
}
