package worker

import (
	"context"

	fashionDto "smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	uc_interfaces "smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/interface/usecase"
	"smart-wardrobe-be/shared/application/constants/eventconstants"
	"smart-wardrobe-be/shared/application/event"
	"smart-wardrobe-be/shared/observability/workerlog"
	"smart-wardrobe-be/shared/pkg/logger"
	"smart-wardrobe-be/shared/presentation/worker"

	"go.uber.org/zap"
)

type WardrobeFashionBindWorker struct {
	*worker.BaseWorker
	messageClient event.IMessageConsumer
	useCase       uc_interfaces.IWardrobeFashionEventUseCase
	logger        logger.Interface
}

func NewWardrobeFashionBindWorker(
	messageClient event.IMessageConsumer,
	useCase uc_interfaces.IWardrobeFashionEventUseCase,
	l logger.Interface,
) *WardrobeFashionBindWorker {
	return &WardrobeFashionBindWorker{
		BaseWorker:    worker.NewBaseWorker("wardrobe_fashion_bind"),
		messageClient: messageClient,
		useCase:       useCase,
		logger:        l,
	}
}

func (w *WardrobeFashionBindWorker) Start(ctx context.Context) {
	w.BaseWorker.Start(ctx)
	if err := event.RegisterTypedConsumer(w.messageClient, eventconstants.QueueWardrobeFashionBind, w.handle, event.ConsumerOpts{}); err != nil {
		w.logger.Error("Failed to register wardrobe fashion bind consumer", zap.Error(err))
	}
}

func (w *WardrobeFashionBindWorker) handle(ctx context.Context, payload fashionDto.FashionItemCreatedEventPayload) error {
	run := workerlog.New("wardrobe_fashion_bind", workerlog.TriggerQueue)
	run.LogReceived(w.logger,
		zap.String("correlationId", event.CorrelationIdFromContext(ctx)),
		zap.String("itemId", payload.ItemID.String()),
		zap.String("fashionItemId", payload.FashionItemID.String()),
	)
	if err := w.useCase.HandleFashionItemCreated(ctx, payload); err != nil {
		run.LogFailure(w.logger, err, zap.String("itemId", payload.ItemID.String()))
		return err
	}
	run.LogSuccessIfData(w.logger, zap.String("itemId", payload.ItemID.String()))
	return nil
}

func (w *WardrobeFashionBindWorker) Stop() {
	w.BaseWorker.Stop()
}
