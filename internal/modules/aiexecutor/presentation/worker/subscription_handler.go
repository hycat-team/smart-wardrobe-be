package worker

import (
	"context"
	"encoding/json"

	aiexecutor_dto "smart-wardrobe-be/internal/modules/aiexecutor/application/dto"
	uc_interfaces "smart-wardrobe-be/internal/modules/aiexecutor/application/interface/usecase"
	shared_msg "smart-wardrobe-be/internal/shared/infrastructure/messaging"
	"smart-wardrobe-be/internal/shared/observability/workerlog"
	"smart-wardrobe-be/pkg/logger"

	"go.uber.org/zap"
)

type PlanChangedWorker struct {
	messageClient shared_msg.IRabbitMQClient
	grantUseCase  uc_interfaces.IGrantUseCase
	logger        logger.Interface
}

func NewPlanChangedWorker(
	messageClient shared_msg.IRabbitMQClient,
	grantUseCase uc_interfaces.IGrantUseCase,
	l logger.Interface,
) *PlanChangedWorker {
	w := &PlanChangedWorker{
		messageClient: messageClient,
		grantUseCase:  grantUseCase,
		logger:        l,
	}
	go w.startConsume()
	return w
}

func (w *PlanChangedWorker) startConsume() {
	ctx := context.Background()
	deliveries, err := w.messageClient.Consume(shared_msg.QueueSubscriptionPlanChanged)
	if err != nil {
		w.logger.Error("[PlanChangedWorker] Failed to consume queue", zap.Error(err))
		return
	}

	go func() {
		for d := range deliveries {
			var payload aiexecutor_dto.PlanChangedPayload
			if err := json.Unmarshal(d.Body, &payload); err != nil {
				w.logger.Error("[PlanChangedWorker] Failed to unmarshal message body", zap.Error(err))
				_ = d.Nack(false, false)
				continue
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
					zap.Error(err),
				)
				_ = d.Nack(false, false)
			} else {
				run.LogSuccess(w.logger,
					zap.String("userId", payload.UserID),
				)
				_ = d.Ack(false)
			}
		}
	}()
}
