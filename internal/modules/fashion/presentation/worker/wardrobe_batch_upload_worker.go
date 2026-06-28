package worker

import (
	"context"

	uc_interfaces "smart-wardrobe-be/internal/modules/fashion/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/application/interface/event"
	"smart-wardrobe-be/internal/shared/observability/workerlog"
	"smart-wardrobe-be/pkg/logger"

	"go.uber.org/zap"
)

type WardrobeBatchUploadWorker struct {
	jobConsumer event.IWardrobeBatchUploadJobConsumer
	useCase     uc_interfaces.IFashionWorkerUseCase
	logger      logger.Interface
}

func NewWardrobeBatchUploadWorker(
	jobConsumer event.IWardrobeBatchUploadJobConsumer,
	useCase uc_interfaces.IFashionWorkerUseCase,
	l logger.Interface,
) *WardrobeBatchUploadWorker {
	w := &WardrobeBatchUploadWorker{
		jobConsumer: jobConsumer,
		useCase:     useCase,
		logger:      l,
	}

	go w.startConsume()

	return w
}

func (w *WardrobeBatchUploadWorker) startConsume() {
	ctx := context.Background()
	err := w.jobConsumer.ConsumeJobs(ctx, func(ctx context.Context, job dto.WardrobeBatchUploadJobDTO) error {
		triggerType := workerlog.TriggerQueue
		if job.RetryCount > 0 {
			triggerType = workerlog.TriggerRequeue
		}
		run := workerlog.New("wardrobe_batch_upload", triggerType)
		run.LogReceived(w.logger,
			zap.String("itemId", job.ItemID.String()),
			zap.String("userId", job.UserID.String()),
			zap.Int("processingVersion", job.ProcessingVersion),
			zap.Int("retryCount", job.RetryCount),
		)
		err := w.useCase.ProcessBackgroundBatchUploadJob(ctx, job, run)
		if err != nil {
			run.LogFailure(w.logger, err,
				zap.String("itemId", job.ItemID.String()),
				zap.String("userId", job.UserID.String()),
				zap.Int("processingVersion", job.ProcessingVersion),
				zap.Int("retryCount", job.RetryCount),
			)
			return err
		}
		run.LogSuccess(w.logger,
			zap.String("itemId", job.ItemID.String()),
			zap.String("userId", job.UserID.String()),
			zap.Int("processingVersion", job.ProcessingVersion),
			zap.Int("retryCount", job.RetryCount),
		)
		return nil
	})

	if err != nil {
		w.logger.Error("Failed to initiate wardrobe batch upload job consumption process", zap.Error(err))
	}
}
