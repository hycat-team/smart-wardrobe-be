package worker

import (
	"context"
	"encoding/json"

	uc_interfaces "smart-wardrobe-be/internal/modules/fashion/application/interface/usecase"
	fashion_contract "smart-wardrobe-be/internal/modules/fashion/contract"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	shared_msg "smart-wardrobe-be/internal/shared/infrastructure/messaging"
	"smart-wardrobe-be/internal/shared/observability/workerlog"
	"smart-wardrobe-be/pkg/logger"

	"go.uber.org/zap"
)

type FashionAnalyzeWorker struct {
	messageClient shared_msg.IRabbitMQClient
	useCase       uc_interfaces.IFashionWorkerUseCase
	logger        logger.Interface
}

func NewFashionAnalyzeWorker(
	messageClient shared_msg.IRabbitMQClient,
	useCase uc_interfaces.IFashionWorkerUseCase,
	l logger.Interface,
) *FashionAnalyzeWorker {
	w := &FashionAnalyzeWorker{
		messageClient: messageClient,
		useCase:       useCase,
		logger:        l,
	}

	go w.startConsume()

	return w
}

func (w *FashionAnalyzeWorker) startConsume() {
	ctx := context.Background()
	deliveries, err := w.messageClient.Consume(shared_msg.QueueFashionAnalyzeItem)
	if err != nil {
		w.logger.Error("Failed to initiate fashion analyze job consumption process", zap.Error(err))
		return
	}

	go func() {
		for d := range deliveries {
			var job fashion_contract.FashionAnalyzeJobDTO
			if err := json.Unmarshal(d.Body, &job); err != nil {
				w.logger.Error("[FashionAnalyzeWorker] Failed to unmarshal message body", zap.Error(err))
				_ = d.Nack(false, false)
				continue
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

			// Map to WardrobeBatchUploadJobDTO to reuse the existing usecase processing logic
			legacyJob := dto.WardrobeBatchUploadJobDTO{
				ItemID:            job.ItemID,
				FashionItemID:     job.FashionItemID,
				UserID:            job.UserID,
				ImageUrl:          job.ImageUrl,
				ImagePublicID:     job.ImagePublicID,
				CategoryID:        job.CategoryID,
				ProcessingVersion: job.ProcessingVersion,
				RetryCount:        job.RetryCount,
				ItemType:          job.ItemType,
			}

			err := w.useCase.ProcessBackgroundBatchUploadJob(ctx, legacyJob, run)
			if err != nil {
				run.LogFailure(w.logger, err,
					zap.String("fashionItemId", job.FashionItemID.String()),
					zap.String("itemId", job.ItemID.String()),
					zap.Error(err),
				)
				_ = d.Nack(false, false)
			} else {
				run.LogSuccess(w.logger,
					zap.String("fashionItemId", job.FashionItemID.String()),
					zap.String("itemId", job.ItemID.String()),
				)
				_ = d.Ack(false)
			}
		}
	}()
}
