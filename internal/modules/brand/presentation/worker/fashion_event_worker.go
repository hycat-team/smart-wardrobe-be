package worker

import (
	"context"

	usecase_interfaces "smart-wardrobe-be/internal/modules/brand/application/interface/usecase"
	fashion_dto "smart-wardrobe-be/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/internal/pkg/logger"
	"smart-wardrobe-be/internal/shared/application/constants/eventconstants"
	"smart-wardrobe-be/internal/shared/application/event"
	"smart-wardrobe-be/internal/shared/observability/workerlog"
	"smart-wardrobe-be/internal/shared/presentation/worker"

	"go.uber.org/zap"
)

type BrandFashionEventWorker struct {
	*worker.BaseWorker
	messageClient event.IMessageConsumer
	useCase       usecase_interfaces.IBrandFashionEventUseCase
	logger        logger.Interface
}

func NewBrandFashionEventWorker(
	messageClient event.IMessageConsumer,
	useCase usecase_interfaces.IBrandFashionEventUseCase,
	l logger.Interface,
) *BrandFashionEventWorker {
	return &BrandFashionEventWorker{
		BaseWorker:    worker.NewBaseWorker("brand_fashion_event"),
		messageClient: messageClient,
		useCase:       useCase,
		logger:        l,
	}
}

func (w *BrandFashionEventWorker) Start(ctx context.Context) {
	w.BaseWorker.Start(ctx)
	if err := event.RegisterTypedConsumer(w.messageClient, eventconstants.QueueBrandFashionAnalyzed, w.handle, event.ConsumerOpts{}); err != nil {
		w.logger.Error("Failed to register brand fashion event consumer", zap.Error(err))
	}
}

func (w *BrandFashionEventWorker) handle(ctx context.Context, payload fashion_dto.FashionAnalyzedEventPayload) error {
	run := workerlog.New("brand_fashion_event", workerlog.TriggerQueue)
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

func (w *BrandFashionEventWorker) Stop() {
	w.BaseWorker.Stop()
}
