package worker

import (
	"context"
	"encoding/json"

	aiexecutor_dto "smart-wardrobe-be/internal/modules/aiexecutor/application/dto"
	uc_interfaces "smart-wardrobe-be/internal/modules/aiexecutor/application/interface/usecase"
	"smart-wardrobe-be/internal/pkg/logger"
	shared_msg "smart-wardrobe-be/internal/shared/infrastructure/messaging"
	"smart-wardrobe-be/internal/shared/observability/workerlog"
	"smart-wardrobe-be/internal/shared/presentation/worker"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

type PlanChangedWorker struct {
	*worker.BaseWorker
	messageClient shared_msg.IRabbitMQClient
	grantUseCase  uc_interfaces.IGrantUseCase
	logger        logger.Interface
}

func NewPlanChangedWorker(
	messageClient shared_msg.IRabbitMQClient,
	grantUseCase uc_interfaces.IGrantUseCase,
	l logger.Interface,
) *PlanChangedWorker {
	return &PlanChangedWorker{
		BaseWorker:    worker.NewBaseWorker("plan_changed"),
		messageClient: messageClient,
		grantUseCase:  grantUseCase,
		logger:        l,
	}
}

func (w *PlanChangedWorker) Start(ctx context.Context) {
	w.BaseWorker.Start(ctx)
	go w.startConsume(ctx)
}

func (w *PlanChangedWorker) startConsume(ctx context.Context) {
	handler := func(d amqp.Delivery) error {
		var payload aiexecutor_dto.PlanChangedPayload
		if err := json.Unmarshal(d.Body, &payload); err != nil {
			w.logger.Error("[PlanChangedWorker] Failed to unmarshal message body", zap.Error(err))
			return err
		}

		run := workerlog.New("plan_changed", workerlog.TriggerQueue)
		run.LogReceived(w.logger,
			zap.String("userId", payload.UserID),
			zap.String("eventType", payload.EventType),
		)

		err := w.grantUseCase.HandlePlanChanged(ctx, payload)
		if err != nil {
			run.LogFailure(w.logger, err,
				zap.String("userId", payload.UserID),
			)
			return err
		}

		run.LogSuccessIfData(w.logger,
			zap.String("userId", payload.UserID),
		)
		return nil
	}

	w.messageClient.RegisterConsumer(shared_msg.QueueSubscriptionPlanChanged, handler)
	w.logger.Info("[PlanChangedWorker] Registered consumer for queue", zap.String("queue", shared_msg.QueueSubscriptionPlanChanged))
}

func (w *PlanChangedWorker) Stop() {
	w.BaseWorker.Stop()
}
