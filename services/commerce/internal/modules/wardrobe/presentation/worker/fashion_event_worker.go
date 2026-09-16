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

type WardrobeFashionEventWorker struct {
	*worker.BaseWorker
	messageClient event.IMessageConsumer
	useCase       uc_interfaces.IWardrobeFashionEventUseCase
	logger        logger.Interface
}

func NewWardrobeFashionEventWorker(
	messageClient event.IMessageConsumer,
	useCase uc_interfaces.IWardrobeFashionEventUseCase,
	l logger.Interface,
) *WardrobeFashionEventWorker {
	return &WardrobeFashionEventWorker{
		BaseWorker:    worker.NewBaseWorker("wardrobe_fashion_event"),
		messageClient: messageClient,
		useCase:       useCase,
		logger:        l,
	}
}

func (w *WardrobeFashionEventWorker) Start(ctx context.Context) {
	w.BaseWorker.Start(ctx)
	if err := event.RegisterTypedConsumer(w.messageClient, eventconstants.QueueWardrobeFashionAnalyzed, w.handle, event.ConsumerOpts{}); err != nil {
		w.logger.Error("Failed to register wardrobe fashion event consumer", zap.Error(err))
	}
}

func (w *WardrobeFashionEventWorker) handle(ctx context.Context, payload fashionDto.FashionAnalyzedEventPayload) error {
	run := workerlog.New("wardrobe_fashion_event", workerlog.TriggerQueue)
	run.LogReceived(w.logger,
		zap.String("correlationId", event.CorrelationIdFromContext(ctx)),
		zap.String("itemId", payload.ItemID.String()),
		zap.String("taskId", payload.TaskID),
		zap.String("status", string(payload.Status)),
	)
	if err := w.useCase.HandleFashionEvent(ctx, payload); err != nil {
		run.LogFailure(w.logger, err,
			zap.String("itemId", payload.ItemID.String()),
			zap.String("taskId", payload.TaskID),
			zap.String("status", string(payload.Status)),
		)
		return err
	}
	run.LogSuccessIfData(w.logger,
		zap.String("itemId", payload.ItemID.String()),
		zap.String("taskId", payload.TaskID),
		zap.String("status", string(payload.Status)),
	)
	return nil
}

func (w *WardrobeFashionEventWorker) Stop() {
	w.BaseWorker.Stop()
}
