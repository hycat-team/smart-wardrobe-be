package worker

import (
	"context"
	"time"

	"smart-wardrobe-be/config"
	uc_interfaces "smart-wardrobe-be/internal/modules/wardrobe/application/interface/usecase"
	"smart-wardrobe-be/pkg/logger"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

// IProcessingRecoveryWorker schedules scans that rescue stale wardrobe processing items.
type IProcessingRecoveryWorker interface {
	Start()
	Stop()
}

// ProcessingRecoveryWorker runs the stale-processing recovery workflow on a cron schedule.
type ProcessingRecoveryWorker struct {
	useCase    uc_interfaces.IWardrobeWorkerUseCase
	cronEngine *cron.Cron
	log        logger.Interface
	cfg        *config.Config
}

// NewProcessingRecoveryWorker builds the processing recovery scheduler.
func NewProcessingRecoveryWorker(
	cfg *config.Config,
	useCase uc_interfaces.IWardrobeWorkerUseCase,
	log logger.Interface,
) IProcessingRecoveryWorker {
	return &ProcessingRecoveryWorker{
		useCase:    useCase,
		cronEngine: cron.New(cron.WithSeconds()),
		log:        log,
		cfg:        cfg,
	}
}

// Start begins the recovery scheduler and executes an initial scan.
func (w *ProcessingRecoveryWorker) Start() {
	go w.executeRecoveryJob()

	_, err := w.cronEngine.AddFunc(w.cfg.Wardrobe.RecoveryScanCron, w.executeRecoveryJob)
	if err != nil {
		w.log.Error("Failed to register wardrobe processing recovery cron task", zap.Error(err))
		return
	}

	w.cronEngine.Start()
}

// Stop stops the recovery scheduler safely.
func (w *ProcessingRecoveryWorker) Stop() {
	if w.cronEngine != nil {
		w.cronEngine.Stop()
	}
}

func (w *ProcessingRecoveryWorker) executeRecoveryJob() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	if err := w.useCase.RecoverStaleProcessingItems(ctx); err != nil {
		w.log.Error("[ProcessingRecoveryWorker] Job failed", zap.Error(err))
	} else {
		w.log.Info("[ProcessingRecoveryWorker] Job succeeded")
	}
}
