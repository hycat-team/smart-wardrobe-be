package worker

import (
	"context"
	"encoding/json"

	"smart-wardrobe-be/internal/modules/fashion/application/dto"
	uc_interfaces "smart-wardrobe-be/internal/modules/fashion/application/interface/usecase"
	"smart-wardrobe-be/internal/pkg/logger"
	shared_msg "smart-wardrobe-be/internal/shared/infrastructure/messaging"
	"smart-wardrobe-be/internal/shared/observability/workerlog"
	"smart-wardrobe-be/internal/shared/presentation/worker"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

type FashionAnalyzeWorker struct {
	*worker.BaseWorker
	messageClient shared_msg.IRabbitMQClient
	useCase       uc_interfaces.IFashionWorkerUseCase
	logger        logger.Interface
}

func NewFashionAnalyzeWorker(
	messageClient shared_msg.IRabbitMQClient,
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
	go w.startConsume(ctx)
}

func (w *FashionAnalyzeWorker) startConsume(ctx context.Context) {
	handler := func(d amqp.Delivery) error {
		var job dto.FashionAnalyzeJobDTO
		if err := json.Unmarshal(d.Body, &job); err != nil {
			w.logger.Error("[FashionAnalyzeWorker] Failed to unmarshal message body", zap.Error(err))
			return err
		}

		triggerType := workerlog.TriggerQueue
		if job.RetryCount > 0 {
			triggerType = workerlog.TriggerRequeue
		}
		run := workerlog.New("fashion_analyze_item", triggerType)
		run.LogReceived(w.logger,
			zap.String("fashionItemId", job.FashionItemID.String()),
			zap.String("itemId", job.ItemID.String()),
			zap.String("itemType", job.ItemType),
			zap.Int("processingVersion", job.ProcessingVersion),
			zap.Int("retryCount", job.RetryCount),
		)

		err := w.useCase.ProcessBackgroundBatchUploadJob(ctx, job, run)
		if err != nil {
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

	w.messageClient.RegisterConsumer(shared_msg.QueueFashionAnalyze, handler)
	w.logger.Info("[FashionAnalyzeWorker] Registered consumer for queue", zap.String("queue", shared_msg.QueueFashionAnalyze))
}
