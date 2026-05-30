package worker

import (
	"context"
	"encoding/json"

	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	uc_interfaces "smart-wardrobe-be/internal/modules/wardrobe/application/interface/usecase"
	"smart-wardrobe-be/internal/shared/infrastructure/rabbitmq"
	"smart-wardrobe-be/pkg/logger"

	"go.uber.org/zap"
)

type BatchCropRabbitMQWorker struct {
	rabbitmqClient rabbitmq.IRabbitMQClient
	useCase        uc_interfaces.IWardrobeUseCase
	logger         logger.Interface
}

func NewBatchCropRabbitMQWorker(
	rabbitmqClient rabbitmq.IRabbitMQClient,
	useCase uc_interfaces.IWardrobeUseCase,
	l logger.Interface,
) *BatchCropRabbitMQWorker {
	w := &BatchCropRabbitMQWorker{
		rabbitmqClient: rabbitmqClient,
		useCase:        useCase,
		logger:         l,
	}

	// Khởi chạy 2 background worker threads lắng nghe RabbitMQ
	for range 2 {
		go w.startConsume()
	}

	return w
}

func (w *BatchCropRabbitMQWorker) startConsume() {
	deliveries, err := w.rabbitmqClient.Consume("batch_crop_jobs")
	if err != nil {
		w.logger.Error("[BatchCropWorker] Failed to start consuming from RabbitMQ queue batch_crop_jobs", zap.Error(err))
		return
	}

	w.logger.Info("[BatchCropWorker] Worker started listening to RabbitMQ queue batch_crop_jobs...")

	for d := range deliveries {
		var job dto.BatchCropJobDTO
		err := json.Unmarshal(d.Body, &job)
		if err != nil {
			w.logger.Error("[BatchCropWorker] Failed to unmarshal message body", zap.Error(err))
			_ = d.Nack(false, false)
			continue
		}

		ctx := context.Background()
		err = w.useCase.ProcessBackgroundCropJob(ctx, job)
		if err != nil {
			w.logger.Error("[BatchCropWorker] Error processing job for item",
				zap.String("itemId", job.ItemID.String()),
				zap.Error(err),
			)
			_ = d.Nack(false, false) // requeue = false, chuyển sang trạng thái Failed
		} else {
			_ = d.Ack(false)
		}
	}
}
