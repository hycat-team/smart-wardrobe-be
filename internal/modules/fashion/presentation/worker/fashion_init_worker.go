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

type FashionInitWorker struct {
	*worker.BaseWorker
	messageClient shared_msg.IRabbitMQClient
	useCase       uc_interfaces.IFashionInitUseCase
	logger        logger.Interface
}

func NewFashionInitWorker(
	messageClient shared_msg.IRabbitMQClient,
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
	go w.startConsume(ctx)
}

func (w *FashionInitWorker) startConsume(ctx context.Context) {
	handler := func(d amqp.Delivery) error {
		var job dto.FashionAnalyzeJobDTO
		if err := json.Unmarshal(d.Body, &job); err != nil {
			w.logger.Error("[FashionInitWorker] Failed to unmarshal message body", zap.Error(err))
			return err
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
			return err
		}
		run.LogSuccessIfData(w.logger,
			zap.String("itemId", job.ItemID.String()),
		)
		return nil
	}

	w.messageClient.RegisterConsumer(shared_msg.QueueFashionInit, handler)
	w.logger.Info("[FashionInitWorker] Registered consumer for queue", zap.String("queue", shared_msg.QueueFashionInit))
}

func (w *FashionInitWorker) Stop() {
	w.BaseWorker.Stop()
}
