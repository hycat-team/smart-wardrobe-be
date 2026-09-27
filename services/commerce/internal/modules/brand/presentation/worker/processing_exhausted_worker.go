package worker

import (
	"context"

	uc_interfaces "smart-wardrobe-be/services/commerce/internal/modules/brand/application/interface/usecase"
	fashionDto "smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/shared/application/constants/eventconstants"
	"smart-wardrobe-be/shared/application/event"
	"smart-wardrobe-be/shared/observability/workerlog"
	"smart-wardrobe-be/shared/pkg/logger"
	"smart-wardrobe-be/shared/presentation/worker"

	"go.uber.org/zap"
)

// BrandProcessingExhaustedWorker consumes processing-exhausted events and moves
// brand items to the terminal `failed` state (FR-034).
type BrandProcessingExhaustedWorker struct {
	*worker.BaseWorker
	messageClient event.IMessageConsumer
	useCase       uc_interfaces.IBrandFashionEventUseCase
	logger        logger.Interface
}

func NewBrandProcessingExhaustedWorker(
	messageClient event.IMessageConsumer,
	useCase uc_interfaces.IBrandFashionEventUseCase,
	l logger.Interface,
) *BrandProcessingExhaustedWorker {
	return &BrandProcessingExhaustedWorker{
		BaseWorker:    worker.NewBaseWorker("brand_processing_exhausted"),
		messageClient: messageClient,
		useCase:       useCase,
		logger:        l,
	}
}

func (w *BrandProcessingExhaustedWorker) Start(ctx context.Context) {
	w.BaseWorker.Start(ctx)
	if err := event.RegisterTypedConsumer(w.messageClient, eventconstants.QueueBrandProcessingExhausted, w.handle, event.ConsumerOpts{}); err != nil {
		w.logger.Error("Failed to register brand processing exhausted consumer", zap.Error(err))
	}
}

func (w *BrandProcessingExhaustedWorker) handle(ctx context.Context, payload fashionDto.FashionProcessingExhaustedEventPayload) error {
	run := workerlog.New("brand_processing_exhausted", workerlog.TriggerQueue)
	run.LogReceived(w.logger,
		zap.String("correlationId", event.CorrelationIdFromContext(ctx)),
		zap.String("itemId", payload.ItemID.String()),
		zap.String("fashionItemId", payload.FashionItemID.String()),
	)
	if err := w.useCase.HandleFashionProcessingExhausted(ctx, payload); err != nil {
		run.LogFailure(w.logger, err, zap.String("itemId", payload.ItemID.String()))
		return err
	}
	run.LogSuccessIfData(w.logger, zap.String("itemId", payload.ItemID.String()))
	return nil
}

func (w *BrandProcessingExhaustedWorker) Stop() {
	w.BaseWorker.Stop()
}
