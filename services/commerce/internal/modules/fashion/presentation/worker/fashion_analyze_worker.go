package worker

import (
	"context"

	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	uc_interfaces "smart-wardrobe-be/services/commerce/internal/modules/fashion/application/interface/usecase"
	"smart-wardrobe-be/shared/application/constants/eventconstants"
	"smart-wardrobe-be/shared/application/event"
	"smart-wardrobe-be/shared/observability/workerlog"
	"smart-wardrobe-be/shared/pkg/logger"
	"smart-wardrobe-be/shared/presentation/worker"

	"go.uber.org/zap"
)

type FashionAnalyzeWorker struct {
	*worker.BaseWorker
	messageClient event.IMessageConsumer
	useCase       uc_interfaces.IFashionWorkerUseCase
	logger        logger.Interface
}

func NewFashionAnalyzeWorker(
	messageClient event.IMessageConsumer,
	useCase uc_interfaces.IFashionWorkerUseCase,
	l logger.Interface,
) *FashionAnalyzeWorker {
	return &FashionAnalyzeWorker{
		BaseWorker:    worker.NewBaseWorker("fashion_analyze"),
		messageClient: messageClient,
		useCase:       useCase,
		logger:        l,
	}
}

func (w *FashionAnalyzeWorker) Start(ctx context.Context) {
	w.BaseWorker.Start(ctx)
	// Concurrency đọc từ Topology registry.
	if err := event.RegisterTypedConsumer(w.messageClient, eventconstants.QueueFashionAnalyze, w.handle, event.ConsumerOpts{}); err != nil {
		w.logger.Error("Failed to register fashion analyze consumer", zap.Error(err))
	}
}

func (w *FashionAnalyzeWorker) handle(ctx context.Context, job dto.FashionAnalyzeJobDTO) error {
	triggerType := workerlog.TriggerQueue
	if job.RetryCount > 0 {
		triggerType = workerlog.TriggerRequeue
	}
	run := workerlog.New("fashion_analyze_item", triggerType)
	run.LogReceived(w.logger,
		zap.String("correlationId", event.CorrelationIdFromContext(ctx)),
		zap.String("fashionItemId", job.FashionItemID.String()),
		zap.String("itemId", job.ItemID.String()),
		zap.String("itemType", job.ItemType),
		zap.Int("processingVersion", job.ProcessingVersion),
		zap.Int("retryCount", job.RetryCount),
	)

	if err := w.useCase.ProcessBackgroundBatchUploadJob(ctx, job, run); err != nil {
		run.LogFailure(w.logger, err,
			zap.String("fashionItemId", job.FashionItemID.String()),
			zap.String("itemId", job.ItemID.String()),
			zap.Error(err),
		)
		return err
	}
	run.LogSuccessIfData(w.logger,
		zap.String("fashionItemId", job.FashionItemID.String()),
		zap.String("itemId", job.ItemID.String()),
	)
	return nil
}
