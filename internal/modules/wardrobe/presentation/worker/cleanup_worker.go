package worker

import (
	"context"

	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	uc_interfaces "smart-wardrobe-be/internal/modules/wardrobe/application/interface/usecase"
	"smart-wardrobe-be/internal/pkg/logger"
	"smart-wardrobe-be/internal/shared/application/constants/eventconstants"
	"smart-wardrobe-be/internal/shared/application/event"
	"smart-wardrobe-be/internal/shared/observability/workerlog"
	"smart-wardrobe-be/internal/shared/presentation/worker"

	"go.uber.org/zap"
)

type WardrobeCleanupWorker struct {
	*worker.BaseWorker
	messageClient event.IMessageConsumer
	useCase       uc_interfaces.IWardrobeFashionEventUseCase
	logger        logger.Interface
}

func NewWardrobeCleanupWorker(
	messageClient event.IMessageConsumer,
	useCase uc_interfaces.IWardrobeFashionEventUseCase,
	l logger.Interface,
) *WardrobeCleanupWorker {
	return &WardrobeCleanupWorker{
		BaseWorker:    worker.NewBaseWorker("wardrobe_cleanup"),
		messageClient: messageClient,
		useCase:       useCase,
		logger:        l,
	}
}

func (w *WardrobeCleanupWorker) Start(ctx context.Context) {
	w.BaseWorker.Start(ctx)
	if err := event.RegisterTypedConsumer(w.messageClient, eventconstants.QueueWardrobeCleanup, w.handle, event.ConsumerOpts{}); err != nil {
		w.logger.Error("Failed to register wardrobe cleanup consumer", zap.Error(err))
	}
}

func (w *WardrobeCleanupWorker) handle(ctx context.Context, payload dto.WardrobeEventPayload) error {
	run := workerlog.New("wardrobe_cleanup", workerlog.TriggerQueue)
	run.LogReceived(w.logger,
		zap.String("correlationId", event.CorrelationIdFromContext(ctx)),
		zap.String("itemId", payload.ItemID.String()),
	)
	if err := w.useCase.HandleWardrobeDeleted(ctx, payload); err != nil {
		run.LogFailure(w.logger, err, zap.String("itemId", payload.ItemID.String()))
		return err
	}
	run.LogSuccessIfData(w.logger, zap.String("itemId", payload.ItemID.String()))
	return nil
}

func (w *WardrobeCleanupWorker) Stop() {
	w.BaseWorker.Stop()
}
