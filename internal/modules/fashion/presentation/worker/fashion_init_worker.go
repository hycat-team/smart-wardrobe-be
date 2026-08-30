package worker

import (
	"context"

	"smart-wardrobe-be/internal/modules/fashion/application/dto"
	uc_interfaces "smart-wardrobe-be/internal/modules/fashion/application/interface/usecase"
	"smart-wardrobe-be/internal/pkg/logger"
	"smart-wardrobe-be/internal/shared/application/constants/eventconstants"
	"smart-wardrobe-be/internal/shared/application/event"
	"smart-wardrobe-be/internal/shared/observability/workerlog"
	"smart-wardrobe-be/internal/shared/presentation/worker"

	"go.uber.org/zap"
)

type FashionInitWorker struct {
	*worker.BaseWorker
	messageClient event.IMessageConsumer
	useCase       uc_interfaces.IFashionInitUseCase
	logger        logger.Interface
}

func NewFashionInitWorker(
	messageClient event.IMessageConsumer,
	useCase uc_interfaces.IFashionInitUseCase,
	l logger.Interface,
) *FashionInitWorker {
	return &FashionInitWorker{
		BaseWorker:    worker.NewBaseWorker("fashion_init"),
		messageClient: messageClient,
		useCase:       useCase,
		logger:        l,
	}
}

func (w *FashionInitWorker) Start(ctx context.Context) {
	w.BaseWorker.Start(ctx)
	if err := event.RegisterTypedConsumer(w.messageClient, eventconstants.QueueFashionInit, w.handle, event.ConsumerOpts{}); err != nil {
		w.logger.Error("Failed to register fashion init consumer", zap.Error(err))
	}
}

func (w *FashionInitWorker) handle(ctx context.Context, job dto.FashionAnalyzeJobDTO) error {
	triggerType := workerlog.TriggerQueue
	if job.RetryCount > 0 {
		triggerType = workerlog.TriggerRequeue
	}
	run := workerlog.New("fashion_init", triggerType)
	run.LogReceived(w.logger,
		zap.String("correlationId", event.CorrelationIdFromContext(ctx)),
		zap.String("itemId", job.ItemID.String()),
		zap.String("itemType", job.ItemType),
	)

	if err := w.useCase.ProcessInitJob(ctx, job, run); err != nil {
		run.LogFailure(w.logger, err,
			zap.String("itemId", job.ItemID.String()),
			zap.Error(err),
		)
		return err
	}
	run.LogSuccessIfData(w.logger,
		zap.String("itemId", job.ItemID.String()),
	)
	return nil
}

func (w *FashionInitWorker) Stop() {
	w.BaseWorker.Stop()
}
