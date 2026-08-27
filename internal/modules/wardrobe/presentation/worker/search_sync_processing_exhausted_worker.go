package worker

import (
	"context"

	fashionDto "smart-wardrobe-be/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/application/interface/event"
	usecase_interfaces "smart-wardrobe-be/internal/modules/wardrobe/application/interface/usecase"
	"smart-wardrobe-be/internal/pkg/logger"
	"smart-wardrobe-be/internal/shared/observability/workerlog"
	"smart-wardrobe-be/internal/shared/presentation/worker"

	"go.uber.org/zap"
)

type ElasticsearchProcessingExhaustedWorker struct {
	*worker.BaseWorker
	eventConsumer event.IElasticsearchProcessingExhaustedEventConsumer
	useCase       usecase_interfaces.ISearchSyncUseCase
	logger        logger.Interface
}

func NewElasticsearchProcessingExhaustedWorker(
	eventConsumer event.IElasticsearchProcessingExhaustedEventConsumer,
	useCase usecase_interfaces.ISearchSyncUseCase,
	l logger.Interface,
) *ElasticsearchProcessingExhaustedWorker {
	return &ElasticsearchProcessingExhaustedWorker{
		BaseWorker:    worker.NewBaseWorker("elasticsearch_processing_exhausted"),
		eventConsumer: eventConsumer,
		useCase:       useCase,
		logger:        l,
	}
}

func (w *ElasticsearchProcessingExhaustedWorker) Start(ctx context.Context) {
	w.BaseWorker.Start(ctx)
	go w.startConsume(ctx)
}

func (w *ElasticsearchProcessingExhaustedWorker) startConsume(ctx context.Context) {
	err := w.eventConsumer.ConsumeEvents(ctx, func(ctx context.Context, payload fashionDto.FashionProcessingExhaustedPayload) error {
		run := workerlog.New("elasticsearch_processing_exhausted", workerlog.TriggerQueue)
		run.LogReceived(w.logger, zap.String("wardrobeItemId", payload.WardrobeItemID.String()))
		if err := w.useCase.HandleFashionProcessingExhausted(ctx, payload, run); err != nil {
			if w.useCase.IsOfflineError(err) {
				run.LogFailure(w.logger, err, zap.String("wardrobeItemId", payload.WardrobeItemID.String()))
				return err
			}
			run.LogFailure(w.logger, err, zap.String("wardrobeItemId", payload.WardrobeItemID.String()))
			return err
		}
		run.LogSuccessIfData(w.logger, zap.String("wardrobeItemId", payload.WardrobeItemID.String()))
		return nil
	})
	if err != nil {
		w.logger.Error("Failed to initiate elasticsearch processing exhausted consumption", zap.Error(err))
	}
}

func (w *ElasticsearchProcessingExhaustedWorker) Stop() {
	w.BaseWorker.Stop()
}
