package worker

import (
	"context"

	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/application/interface/event"
	usecase_interfaces "smart-wardrobe-be/internal/modules/wardrobe/application/interface/usecase"
	"smart-wardrobe-be/internal/pkg/logger"
	"smart-wardrobe-be/internal/shared/observability/workerlog"
	"smart-wardrobe-be/internal/shared/presentation/worker"

	"go.uber.org/zap"
)

type WardrobeCleanupWorker struct {
	*worker.BaseWorker
	eventConsumer event.IWardrobeCleanupEventConsumer
	useCase       usecase_interfaces.IWardrobeFashionEventUseCase
	logger        logger.Interface
}

func NewWardrobeCleanupWorker(
	eventConsumer event.IWardrobeCleanupEventConsumer,
	useCase usecase_interfaces.IWardrobeFashionEventUseCase,
	l logger.Interface,
) *WardrobeCleanupWorker {
	return &WardrobeCleanupWorker{
		BaseWorker:    worker.NewBaseWorker("wardrobe_cleanup"),
		eventConsumer: eventConsumer,
		useCase:       useCase,
		logger:        l,
	}
}

func (w *WardrobeCleanupWorker) Start(ctx context.Context) {
	w.BaseWorker.Start(ctx)
	go w.startConsume(ctx)
}

func (w *WardrobeCleanupWorker) startConsume(ctx context.Context) {
	err := w.eventConsumer.ConsumeEvents(ctx, func(ctx context.Context, payload dto.WardrobeEventPayload) error {
		run := workerlog.New("wardrobe_cleanup", workerlog.TriggerQueue)
		run.LogReceived(w.logger, zap.String("itemId", payload.ItemID.String()))
		if err := w.useCase.HandleWardrobeDeleted(ctx, payload); err != nil {
			run.LogFailure(w.logger, err, zap.String("itemId", payload.ItemID.String()))
			return err
		}
		run.LogSuccessIfData(w.logger, zap.String("itemId", payload.ItemID.String()))
		return nil
	})
	if err != nil {
		w.logger.Error("Failed to initiate wardrobe cleanup consumption", zap.Error(err))
	}
}

func (w *WardrobeCleanupWorker) Stop() {
	w.BaseWorker.Stop()
}
