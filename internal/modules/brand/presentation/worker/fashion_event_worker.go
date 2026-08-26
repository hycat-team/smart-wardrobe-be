package worker

import (
	"context"

	brand_event "smart-wardrobe-be/internal/modules/brand/application/interface/event"
	usecase_interfaces "smart-wardrobe-be/internal/modules/brand/application/interface/usecase"
	fashion_dto "smart-wardrobe-be/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/internal/pkg/logger"
	"smart-wardrobe-be/internal/shared/application/constants/analyzetaskstatus"
	"smart-wardrobe-be/internal/shared/observability/workerlog"
	"smart-wardrobe-be/internal/shared/presentation/worker"

	"go.uber.org/zap"
)

type BrandFashionEventWorker struct {
	*worker.BaseWorker
	eventConsumer brand_event.IBrandFashionAnalyzedEventConsumer
	useCase       usecase_interfaces.IBrandFashionEventUseCase
	logger        logger.Interface
}

func NewBrandFashionEventWorker(
	eventConsumer brand_event.IBrandFashionAnalyzedEventConsumer,
	useCase usecase_interfaces.IBrandFashionEventUseCase,
	l logger.Interface,
) *BrandFashionEventWorker {
	return &BrandFashionEventWorker{
		BaseWorker:    worker.NewBaseWorker("brand_fashion_event"),
		eventConsumer: eventConsumer,
		useCase:       useCase,
		logger:        l,
	}
}

func (w *BrandFashionEventWorker) Start(ctx context.Context) {
	w.BaseWorker.Start(ctx)
	go w.startConsume(ctx)
}

func (w *BrandFashionEventWorker) startConsume(ctx context.Context) {
	err := w.eventConsumer.ConsumeEvents(ctx, func(ctx context.Context, payload fashion_dto.FashionAnalyzedEventPayload) error {
		run := workerlog.New("brand_fashion_event", workerlog.TriggerQueue)
		run.LogReceived(w.logger,
			zap.String("itemId", payload.ItemID.String()),
			zap.String("taskId", payload.TaskID),
			zap.String("status", string(payload.Status)),
		)

		var err error
		if payload.Status == analyzetaskstatus.Failed {
			err = w.useCase.HandleFashionFailed(ctx, payload)
		} else {
			err = w.useCase.HandleFashionAnalyzed(ctx, payload)
		}

		if err != nil {
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
	})

	if err != nil {
		w.logger.Error("Failed to initiate brand fashion event consumption process", zap.Error(err))
	}
}

func (w *BrandFashionEventWorker) Stop() {
	w.BaseWorker.Stop()
}
