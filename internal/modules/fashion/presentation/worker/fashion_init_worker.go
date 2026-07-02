package worker

import (
	"context"
	"encoding/json"

	uc_interfaces "smart-wardrobe-be/internal/modules/fashion/application/interface/usecase"
	fashion_contract "smart-wardrobe-be/internal/modules/fashion/contract"
	shared_msg "smart-wardrobe-be/internal/shared/infrastructure/messaging"
	"smart-wardrobe-be/internal/shared/observability/workerlog"
	"smart-wardrobe-be/pkg/logger"

	"go.uber.org/zap"
)

type FashionInitWorker struct {
	messageClient shared_msg.IRabbitMQClient
	useCase       uc_interfaces.IFashionInitUseCase
	logger        logger.Interface
}

func NewFashionInitWorker(
	messageClient shared_msg.IRabbitMQClient,
	useCase uc_interfaces.IFashionInitUseCase,
	l logger.Interface,
) *FashionInitWorker {
	w := &FashionInitWorker{
		messageClient: messageClient,
		useCase:       useCase,
		logger:        l,
	}

	go w.startConsume()

	return w
}

func (w *FashionInitWorker) startConsume() {
	ctx := context.Background()
	deliveries, err := w.messageClient.Consume(shared_msg.QueueFashionInit)
	if err != nil {
		w.logger.Error("[FashionInitWorker] Failed to consume queue", zap.Error(err))
		return
	}

	go func() {
		for d := range deliveries {
			var job fashion_contract.FashionAnalyzeJobDTO
			if err := json.Unmarshal(d.Body, &job); err != nil {
				w.logger.Error("[FashionInitWorker] Failed to unmarshal message body", zap.Error(err))
				_ = d.Nack(false, false)
				continue
			}

			triggerType := workerlog.TriggerQueue
			if job.RetryCount > 0 {
				triggerType = workerlog.TriggerRequeue
			}
			run := workerlog.New("fashion_init", triggerType)
			run.LogReceived(w.logger,
				zap.String("itemId", job.ItemID.String()),
				zap.String("itemType", job.ItemType),
			)

			err := w.useCase.ProcessInitJob(ctx, job, run)
			if err != nil {
				run.LogFailure(w.logger, err,
					zap.String("itemId", job.ItemID.String()),
					zap.Error(err),
				)
				_ = d.Nack(false, false)
			} else {
				run.LogSuccess(w.logger,
					zap.String("itemId", job.ItemID.String()),
				)
				_ = d.Ack(false)
			}
		}
	}()
}
