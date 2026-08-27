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

type WardrobeFashionBindWorker struct {
	*worker.BaseWorker
	eventConsumer event.IWardrobeFashionBindEventConsumer
	useCase       usecase_interfaces.IWardrobeFashionEventUseCase
	logger        logger.Interface
}

func NewWardrobeFashionBindWorker(
	eventConsumer event.IWardrobeFashionBindEventConsumer,
	useCase usecase_interfaces.IWardrobeFashionEventUseCase,
	l logger.Interface,
) *WardrobeFashionBindWorker {
	return &WardrobeFashionBindWorker{
		BaseWorker:    worker.NewBaseWorker("wardrobe_fashion_bind"),
		eventConsumer: eventConsumer,
		useCase:       useCase,
		logger:        l,
	}
}

func (w *WardrobeFashionBindWorker) Start(ctx context.Context) {
	w.BaseWorker.Start(ctx)
	go w.startConsume(ctx)
}

func (w *WardrobeFashionBindWorker) startConsume(ctx context.Context) {
	err := w.eventConsumer.ConsumeEvents(ctx, func(ctx context.Context, payload fashionDto.FashionItemCreatedPayload) error {
		run := workerlog.New("wardrobe_fashion_bind", workerlog.TriggerQueue)
		run.LogReceived(w.logger,
			zap.String("itemId", payload.ItemID.String()),
			zap.String("fashionItemId", payload.FashionItemID.String()),
		)
		if err := w.useCase.HandleFashionItemCreated(ctx, payload); err != nil {
			run.LogFailure(w.logger, err, zap.String("itemId", payload.ItemID.String()))
			return err
		}
		run.LogSuccessIfData(w.logger, zap.String("itemId", payload.ItemID.String()))
		return nil
	})
	if err != nil {
		w.logger.Error("Failed to initiate wardrobe fashion bind consumption", zap.Error(err))
	}
}

func (w *WardrobeFashionBindWorker) Stop() {
	w.BaseWorker.Stop()
}
