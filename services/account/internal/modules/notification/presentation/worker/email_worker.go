package worker

import (
	"context"

	"smart-wardrobe-be/services/account/internal/modules/notification/application/dto"
	"smart-wardrobe-be/services/account/internal/modules/notification/application/interface/usecase"
	"smart-wardrobe-be/shared/application/constants/eventconstants"
	"smart-wardrobe-be/shared/application/event"
	"smart-wardrobe-be/shared/config"
	"smart-wardrobe-be/shared/observability/workerlog"
	"smart-wardrobe-be/shared/pkg/logger"
	"smart-wardrobe-be/shared/presentation/worker"

	"go.uber.org/zap"
)

type EmailWorker struct {
	*worker.BaseWorker
	messageClient event.IMessageConsumer
	emailUC       usecase.IEmailUseCase
	emailEnabled  bool
	logger        logger.Interface
}

func NewEmailWorker(
	messageClient event.IMessageConsumer,
	emailUC usecase.IEmailUseCase,
	cfg *config.Config,
	l logger.Interface,
) *EmailWorker {
	return &EmailWorker{
		BaseWorker:    worker.NewBaseWorker("email"),
		messageClient: messageClient,
		emailUC:       emailUC,
		emailEnabled:  cfg.Email.Enabled,
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

	// 025 T113: `email.enabled: false` makes this handler consume the message and
	// acknowledge it without delivering it. The consumer is deliberately still
	// registered and still acks, rather than nacking, requeueing or unregistering:
	//
	//   - nacking/requeueing keeps the message on the broker, so the ~125 rows a
	//     bulk verification run leaves behind would be delivered the moment the
	//     switch is turned back on — a later single-account run would send 126
	//     messages instead of one;
	//   - not registering the consumer at all has the same effect one layer down,
	//     because the queue then accumulates instead of draining.
	//
	// Acking while disabled is what makes the backlog unreleasable: each row is
	// marked handled at the moment it is consumed during the disabled window.
	// Those specific emails are lost on purpose — this switch exists to discard
	// test traffic, and a grant that produced test traffic was never a real
	// grant — and the alternative, holding them, is a queue that fires later.
	if !w.emailEnabled {
		run.AddTotal(1)
		run.AddSkipped(1)
		run.ChildInfo(w.logger, "Email delivery skipped: outbound email is disabled (email.enabled=false)",
			zap.String("to", payload.To),
			zap.String("subject", payload.Subject),
		)
		run.LogSuccessIfData(w.logger)
		return nil
	}

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
