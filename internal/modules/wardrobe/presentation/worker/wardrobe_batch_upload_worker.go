package worker

import (
	"context"

	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/application/interface/event"
	uc_interfaces "smart-wardrobe-be/internal/modules/wardrobe/application/interface/usecase"
	"smart-wardrobe-be/pkg/logger"

	"go.uber.org/zap"
)

type WardrobeBatchUploadWorker struct {
	jobConsumer event.IWardrobeBatchUploadJobConsumer
	useCase     uc_interfaces.IWardrobeWorkerUseCase
	logger      logger.Interface
}

func NewWardrobeBatchUploadWorker(
	jobConsumer event.IWardrobeBatchUploadJobConsumer,
	useCase uc_interfaces.IWardrobeWorkerUseCase,
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
		return w.useCase.ProcessBackgroundBatchUploadJob(ctx, job)
	})

	if err != nil {
		w.logger.Error("Failed to initiate wardrobe batch upload job consumption process", zap.Error(err))
	} else {
		w.logger.Info("Wardrobe batch upload coordinator successfully registered job handling callback")
	}
}
