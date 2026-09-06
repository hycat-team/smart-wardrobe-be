package worker

import (
	"context"

	aiexecutor_dto "smart-wardrobe-be/internal/modules/aiexecutor/application/dto"
	uc_interfaces "smart-wardrobe-be/internal/modules/aiexecutor/application/interface/usecase"
	"smart-wardrobe-be/internal/pkg/logger"
	"smart-wardrobe-be/internal/shared/application/constants/eventconstants"
	"smart-wardrobe-be/internal/shared/application/event"
	"smart-wardrobe-be/internal/shared/observability/workerlog"
	"smart-wardrobe-be/internal/shared/presentation/worker"

	"go.uber.org/zap"
)

type PlanChangedWorker struct {
	*worker.BaseWorker
	messageClient event.IMessageConsumer
	grantUseCase  uc_interfaces.IGrantUseCase
	logger        logger.Interface
}

func NewPlanChangedWorker(
	messageClient event.IMessageConsumer,
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
	if err := event.RegisterTypedConsumer(w.messageClient, eventconstants.QueueSubscriptionPlanChanged, w.handle, event.ConsumerOpts{}); err != nil {
		w.logger.Error("Failed to register plan changed consumer", zap.Error(err))
	}
}

func (w *PlanChangedWorker) handle(ctx context.Context, payload aiexecutor_dto.PlanChangedEventPayload) error {
	run := workerlog.New("plan_changed", workerlog.TriggerQueue)
	run.LogReceived(w.logger,
		zap.String("correlationId", event.CorrelationIdFromContext(ctx)),
		zap.String("userId", payload.UserID),
		zap.String("eventType", payload.EventType),
	)

	if err := w.grantUseCase.HandlePlanChanged(ctx, payload); err != nil {
		run.LogFailure(w.logger, err,
			zap.String("userId", payload.UserID),
		)
		return err
	}

	run.AddTotal(1)
	run.AddSuccess(1)
	run.LogSuccessIfData(w.logger,
		zap.String("userId", payload.UserID),
	)
	return nil
}

func (w *PlanChangedWorker) Stop() {
	w.BaseWorker.Stop()
}
