package worker

import (
	"context"

	fashionDto "smart-wardrobe-be/internal/modules/fashion/application/dto"
	uc_interfaces "smart-wardrobe-be/internal/modules/wardrobe/application/interface/usecase"
	"smart-wardrobe-be/internal/pkg/logger"
	"smart-wardrobe-be/internal/shared/application/constants/eventconstants"
	"smart-wardrobe-be/internal/shared/application/event"
	"smart-wardrobe-be/internal/shared/observability/workerlog"
	"smart-wardrobe-be/internal/shared/presentation/worker"

	"go.uber.org/zap"
)

type WardrobeProcessingExhaustedWorker struct {
	*worker.BaseWorker
	messageClient event.IMessageConsumer
	useCase       uc_interfaces.IWardrobeFashionEventUseCase
	logger        logger.Interface
}

func NewWardrobeProcessingExhaustedWorker(
	messageClient event.IMessageConsumer,
	useCase uc_interfaces.IWardrobeFashionEventUseCase,
	l logger.Interface,
) *WardrobeProcessingExhaustedWorker {
	return &WardrobeProcessingExhaustedWorker{
		BaseWorker:    worker.NewBaseWorker("wardrobe_processing_exhausted"),
		messageClient: messageClient,
		useCase:       useCase,
		logger:        l,
	}
}

func (w *WardrobeProcessingExhaustedWorker) Start(ctx context.Context) {
	w.BaseWorker.Start(ctx)
	if err := event.RegisterTypedConsumer(w.messageClient, eventconstants.QueueWardrobeProcessingExhausted, w.handle, event.ConsumerOpts{}); err != nil {
		w.logger.Error("Failed to register wardrobe processing exhausted consumer", zap.Error(err))
	}
}

func (w *WardrobeProcessingExhaustedWorker) handle(ctx context.Context, payload fashionDto.FashionProcessingExhaustedEventPayload) error {
	run := workerlog.New("wardrobe_processing_exhausted", workerlog.TriggerQueue)
	run.LogReceived(w.logger,
		zap.String("correlationId", event.CorrelationIdFromContext(ctx)),
		zap.String("wardrobeItemId", payload.WardrobeItemID.String()),
		zap.String("fashionItemId", payload.FashionItemID.String()),
	)
	if err := w.useCase.HandleFashionProcessingExhausted(ctx, payload); err != nil {
		run.LogFailure(w.logger, err, zap.String("wardrobeItemId", payload.WardrobeItemID.String()))
		return err
	}
	run.LogSuccessIfData(w.logger, zap.String("wardrobeItemId", payload.WardrobeItemID.String()))
	return nil
}

func (w *WardrobeProcessingExhaustedWorker) Stop() {
	w.BaseWorker.Stop()
}
