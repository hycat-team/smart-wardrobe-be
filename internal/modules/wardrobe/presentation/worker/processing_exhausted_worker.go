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

type WardrobeProcessingExhaustedWorker struct {
	*worker.BaseWorker
	eventConsumer event.IWardrobeProcessingExhaustedEventConsumer
	useCase       usecase_interfaces.IWardrobeFashionEventUseCase
	logger        logger.Interface
}

func NewWardrobeProcessingExhaustedWorker(
	eventConsumer event.IWardrobeProcessingExhaustedEventConsumer,
	useCase usecase_interfaces.IWardrobeFashionEventUseCase,
	l logger.Interface,
) *WardrobeProcessingExhaustedWorker {
	return &WardrobeProcessingExhaustedWorker{
		BaseWorker:    worker.NewBaseWorker("wardrobe_processing_exhausted"),
		eventConsumer: eventConsumer,
		useCase:       useCase,
		logger:        l,
	}
}

func (w *WardrobeProcessingExhaustedWorker) Start(ctx context.Context) {
	w.BaseWorker.Start(ctx)
	go w.startConsume(ctx)
}

func (w *WardrobeProcessingExhaustedWorker) startConsume(ctx context.Context) {
	err := w.eventConsumer.ConsumeEvents(ctx, func(ctx context.Context, payload fashionDto.FashionProcessingExhaustedPayload) error {
		run := workerlog.New("wardrobe_processing_exhausted", workerlog.TriggerQueue)
		run.LogReceived(w.logger,
			zap.String("wardrobeItemId", payload.WardrobeItemID.String()),
			zap.String("fashionItemId", payload.FashionItemID.String()),
		)
		if err := w.useCase.HandleFashionProcessingExhausted(ctx, payload); err != nil {
			run.LogFailure(w.logger, err, zap.String("wardrobeItemId", payload.WardrobeItemID.String()))
			return err
		}
		run.LogSuccessIfData(w.logger, zap.String("wardrobeItemId", payload.WardrobeItemID.String()))
		return nil
	})
	if err != nil {
		w.logger.Error("Failed to initiate wardrobe processing exhausted consumption", zap.Error(err))
	}
}

func (w *WardrobeProcessingExhaustedWorker) Stop() {
	w.BaseWorker.Stop()
}
