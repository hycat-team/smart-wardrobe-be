package worker

import (
	"context"

	usecase_interfaces "smart-wardrobe-be/services/commerce/internal/modules/brand/application/interface/usecase"
	fashion_dto "smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/shared/application/constants/eventconstants"
	"smart-wardrobe-be/shared/application/event"
	"smart-wardrobe-be/shared/observability/workerlog"
	"smart-wardrobe-be/shared/pkg/logger"
	"smart-wardrobe-be/shared/presentation/worker"

	"go.uber.org/zap"
)

type BrandFashionBindWorker struct {
	*worker.BaseWorker
	messageClient event.IMessageConsumer
	useCase       usecase_interfaces.IBrandFashionEventUseCase
	logger        logger.Interface
}

func NewBrandFashionBindWorker(
	messageClient event.IMessageConsumer,
	useCase usecase_interfaces.IBrandFashionEventUseCase,
	l logger.Interface,
) *BrandFashionBindWorker {
	return &BrandFashionBindWorker{
		BaseWorker:    worker.NewBaseWorker("brand_fashion_bind"),
		messageClient: messageClient,
		useCase:       useCase,
		logger:        l,
	}
}

func (w *BrandFashionBindWorker) Start(ctx context.Context) {
	w.BaseWorker.Start(ctx)
	if err := event.RegisterTypedConsumer(w.messageClient, eventconstants.QueueBrandFashionBind, w.handle, event.ConsumerOpts{}); err != nil {
		w.logger.Error("Failed to register brand fashion bind consumer", zap.Error(err))
	}
}

func (w *BrandFashionBindWorker) handle(ctx context.Context, payload fashion_dto.FashionItemCreatedEventPayload) error {
	run := workerlog.New("brand_fashion_bind", workerlog.TriggerQueue)
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

func (w *BrandFashionBindWorker) Stop() {
	w.BaseWorker.Stop()
}
