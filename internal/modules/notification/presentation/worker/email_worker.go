package worker

import (
	"context"

	"smart-wardrobe-be/internal/modules/notification/application/dto"
	"smart-wardrobe-be/internal/modules/notification/application/interface/usecase"
	"smart-wardrobe-be/internal/pkg/logger"
	"smart-wardrobe-be/internal/shared/application/constants/eventconstants"
	"smart-wardrobe-be/internal/shared/application/event"
	"smart-wardrobe-be/internal/shared/observability/workerlog"
	"smart-wardrobe-be/internal/shared/presentation/worker"

	"go.uber.org/zap"
)

type EmailWorker struct {
	*worker.BaseWorker
	messageClient event.IMessageConsumer
	emailUC       usecase.IEmailUseCase
	logger        logger.Interface
}

func NewEmailWorker(
	messageClient event.IMessageConsumer,
	emailUC usecase.IEmailUseCase,
	l logger.Interface,
) *EmailWorker {
	return &EmailWorker{
		BaseWorker:    worker.NewBaseWorker("email"),
		messageClient: messageClient,
		emailUC:       emailUC,
		logger:        l,
	}
}

func (w *EmailWorker) Start(ctx context.Context) {
	w.BaseWorker.Start(ctx)
	if err := event.RegisterTypedConsumer(w.messageClient, eventconstants.QueueNotificationEmail, w.handle, event.ConsumerOpts{}); err != nil {
		w.logger.Error("Failed to register email consumer", zap.Error(err))
	}
}

func (w *EmailWorker) handle(ctx context.Context, payload dto.EmailEventPayload) error {
	run := workerlog.New("email", workerlog.TriggerQueue)
	run.LogReceived(w.logger,
		zap.String("correlationId", event.CorrelationIdFromContext(ctx)),
		zap.String("to", payload.To),
		zap.String("subject", payload.Subject),
	)

	if err := w.emailUC.SendEmail(ctx, payload.To, payload.Subject, payload.Body); err != nil {
		run.LogFailure(w.logger, err,
			zap.String("to", payload.To),
			zap.String("subject", payload.Subject),
		)
		return err
	}

	run.AddTotal(1)
	run.AddSuccess(1)
	run.LogSuccessIfData(w.logger,
		zap.String("to", payload.To),
		zap.String("subject", payload.Subject),
	)
	return nil
}

func (w *EmailWorker) Stop() {
	w.BaseWorker.Stop()
}
