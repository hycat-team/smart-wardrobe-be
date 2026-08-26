package worker

import (
	"context"

	fashionDto "smart-wardrobe-be/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/application/interface/event"
	usecase_interfaces "smart-wardrobe-be/internal/modules/wardrobe/application/interface/usecase"
	"smart-wardrobe-be/internal/pkg/logger"
	"smart-wardrobe-be/internal/shared/application/constants/analyzetaskstatus"
	"smart-wardrobe-be/internal/shared/observability/workerlog"
	"smart-wardrobe-be/internal/shared/presentation/worker"

	"go.uber.org/zap"
)

type WardrobeFashionEventWorker struct {
	*worker.BaseWorker
	eventConsumer event.IWardrobeFashionAnalyzedEventConsumer
	useCase       usecase_interfaces.IWardrobeFashionEventUseCase
	logger        logger.Interface
}

func NewWardrobeFashionEventWorker(
	eventConsumer event.IWardrobeFashionAnalyzedEventConsumer,
	useCase usecase_interfaces.IWardrobeFashionEventUseCase,
	l logger.Interface,
) *WardrobeFashionEventWorker {
	return &WardrobeFashionEventWorker{
		BaseWorker:    worker.NewBaseWorker("wardrobe_fashion_event"),
		eventConsumer: eventConsumer,
		useCase:       useCase,
		logger:        l,
	}
}

func (w *WardrobeFashionEventWorker) Start(ctx context.Context) {
	w.BaseWorker.Start(ctx)
	go w.startConsume(ctx)
}

func (w *WardrobeFashionEventWorker) startConsume(ctx context.Context) {
	err := w.eventConsumer.ConsumeEvents(ctx, func(ctx context.Context, payload fashionDto.FashionAnalyzedEventPayload) error {
		run := workerlog.New("wardrobe_fashion_event", workerlog.TriggerQueue)
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
		w.logger.Error("Failed to initiate wardrobe fashion event consumption process", zap.Error(err))
	}
}

func (w *WardrobeFashionEventWorker) Stop() {
	w.BaseWorker.Stop()
}
