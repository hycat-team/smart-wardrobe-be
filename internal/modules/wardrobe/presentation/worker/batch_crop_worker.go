package worker

import (
	"context"

	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/application/interface/event"
	uc_interfaces "smart-wardrobe-be/internal/modules/wardrobe/application/interface/usecase"
	"smart-wardrobe-be/pkg/logger"

	"go.uber.org/zap"
)

type BatchCropWorker struct {
	jobConsumer event.IBatchCropJobConsumer
	useCase     uc_interfaces.IWardrobeUseCase
	logger      logger.Interface
}

func NewBatchCropWorker(
	jobConsumer event.IBatchCropJobConsumer,
	useCase uc_interfaces.IWardrobeUseCase,
	l logger.Interface,
) *BatchCropWorker {
	w := &BatchCropWorker{
		jobConsumer: jobConsumer,
		useCase:     useCase,
		logger:      l,
	}

	// Khởi chạy việc lắng nghe hàng đợi công việc thông qua Consumer của tầng Application
	go w.startConsume()

	return w
}

func (w *BatchCropWorker) startConsume() {
	ctx := context.Background()
	err := w.jobConsumer.ConsumeJobs(ctx, func(ctx context.Context, job dto.BatchCropJobDTO) error {
		return w.useCase.ProcessBackgroundCropJob(ctx, job)
	})

	if err != nil {
		w.logger.Error("Failed to initiate batch crop job consumption process", zap.Error(err))
	} else {
		w.logger.Info("Batch crop coordinator successfully registered job handling callback")
	}
}
