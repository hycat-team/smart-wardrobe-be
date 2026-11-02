package worker

import (
	"context"
	"encoding/json"
	"time"

	"smart-wardrobe-be/services/account/internal/modules/notification/application/dto"
	"smart-wardrobe-be/services/account/internal/modules/notification/application/interface/usecase"
	notificationerrors "smart-wardrobe-be/services/account/internal/modules/notification/domain/errors"
	"smart-wardrobe-be/shared/application/constants/eventconstants"
	"smart-wardrobe-be/shared/application/event"
	"smart-wardrobe-be/shared/config"
	"smart-wardrobe-be/shared/observability/workerlog"
	"smart-wardrobe-be/shared/pkg/logger"
	"smart-wardrobe-be/shared/presentation/worker"

	"go.uber.org/zap"
)

const (
	// maxEmailDeliveryAttempts counts DEFERRALS, not sends: the first attempt is
	// free, then the message goes to the delay queue at most this many times.
	// Four deferrals at five minutes each span about twenty minutes, which covers
	// a relay restart without leaving the customer without the email for an hour.
	maxEmailDeliveryAttempts = 4

	// emailDeliveryDedupeTTL comfortably outlives the broker's redelivery window,
	// so a redelivery caused by a lost ack is recognised as the same delivery.
	emailDeliveryDedupeTTL = 24 * time.Hour
)

type EmailWorker struct {
	*worker.BaseWorker
	messageClient  event.IMessageConsumer
	emailUC        usecase.IEmailUseCase
	retryPublisher event.IRawMessagePublisher
	dedupe         usecase.IDeliveryDeduplicator
	emailEnabled   bool
	logger         logger.Interface
}

func NewEmailWorker(
	messageClient event.IMessageConsumer,
	emailUC usecase.IEmailUseCase,
	retryPublisher event.IRawMessagePublisher,
	dedupe usecase.IDeliveryDeduplicator,
	cfg *config.Config,
	l logger.Interface,
) *EmailWorker {
	return &EmailWorker{
		BaseWorker:     worker.NewBaseWorker("email"),
		messageClient:  messageClient,
		emailUC:        emailUC,
		retryPublisher: retryPublisher,
		dedupe:         dedupe,
		emailEnabled:   cfg.Email.Enabled,
		logger:         l,
	}
}

func (w *EmailWorker) Start(ctx context.Context) {
	w.BaseWorker.Start(ctx)
	if !w.emailEnabled {
		// Loud on purpose. `email.enabled=false` makes this handler acknowledge
		// and DROP every message, and the alternative to this line is an operator
		// who believes transactional email is flowing while it is being discarded.
		w.logger.Warn("Outbound email is DISABLED: every notification will be acknowledged and discarded (email.enabled=false)")
	}
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
		zap.String("outboxId", payload.OutboxID),
		zap.Int("retryCount", payload.RetryCount),
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

	if w.alreadyDelivered(ctx, payload) {
		run.AddTotal(1)
		run.AddSkipped(1)
		run.ChildInfo(w.logger, "Email delivery skipped: this outbox row was already delivered",
			zap.String("outboxId", payload.OutboxID),
			zap.String("to", payload.To),
		)
		run.LogSuccessIfData(w.logger)
		return nil
	}

	if err := w.emailUC.SendEmail(ctx, payload.To, payload.Subject, payload.Body); err != nil {
		if notificationerrors.IsTransient(err) && payload.RetryCount < maxEmailDeliveryAttempts {
			return w.deferDelivery(ctx, run, payload, err)
		}

		run.LogFailure(w.logger, err,
			zap.String("to", payload.To),
			zap.String("subject", payload.Subject),
			zap.String("outboxId", payload.OutboxID),
			zap.Int("retryCount", payload.RetryCount),
			zap.Bool("transient", notificationerrors.IsTransient(err)),
		)
		// Permanent, or out of attempts: dead-letter it. The message stays
		// inspectable in notification_email_queue.dlq instead of being dropped.
		return event.PermanentError{Err: err}
	}

	w.rememberDelivered(ctx, payload)

	run.AddTotal(1)
	run.AddSuccess(1)
	run.LogSuccessIfData(w.logger,
		zap.String("to", payload.To),
		zap.String("subject", payload.Subject),
		zap.String("outboxId", payload.OutboxID),
	)
	return nil
}

// deferDelivery hands the message to the delay queue and ACKNOWLEDGES the one we
// are holding.
//
// Publish-then-ack, never the other way round: if we ack first and the publish
// then fails, the email is gone. If the process dies between the two, the broker
// redelivers and the dedupe key keeps the customer from seeing it twice.
func (w *EmailWorker) deferDelivery(ctx context.Context, run *workerlog.Run, payload dto.EmailEventPayload, sendErr error) error {
	if w.retryPublisher == nil {
		run.LogFailure(w.logger, sendErr,
			zap.String("outboxId", payload.OutboxID),
			zap.String("reason", "no retry publisher wired"),
		)
		return event.PermanentError{Err: sendErr}
	}

	next := payload
	next.RetryCount++
	raw, err := json.Marshal(next)
	if err != nil {
		run.LogFailure(w.logger, err, zap.String("outboxId", payload.OutboxID))
		return event.PermanentError{Err: err}
	}

	headers := map[string]any{"x-retry-count": next.RetryCount}
	if correlationID := event.CorrelationIdFromContext(ctx); correlationID != "" {
		headers["x-correlation-id"] = correlationID
	}

	if err := w.retryPublisher.PublishDirect(ctx, eventconstants.TopicNotificationEmailRetry, raw, headers); err != nil {
		// The broker refused a publish. Requeueing would spin (this queue is
		// classic, so there is no delivery limit to stop it), so the message goes to
		// the DLQ where it can be replayed by hand once the broker is healthy.
		run.LogFailure(w.logger, err,
			zap.String("outboxId", payload.OutboxID),
			zap.String("reason", "could not publish to the retry queue"),
		)
		return event.PermanentError{Err: err}
	}

	run.AddTotal(1)
	run.AddSkipped(1)
	run.ChildInfo(w.logger, "Email delivery deferred to the retry queue",
		zap.String("to", payload.To),
		zap.String("subject", payload.Subject),
		zap.String("outboxId", payload.OutboxID),
		zap.Int("retryCount", next.RetryCount),
		zap.Error(sendErr),
	)
	run.LogSuccessIfData(w.logger)
	return nil
}

// alreadyDelivered answers "has this exact outbox row been sent". A miss on Redis
// must not stop the email: the dedupe is an optimisation against duplicates, not
// a gate on delivery.
func (w *EmailWorker) alreadyDelivered(ctx context.Context, payload dto.EmailEventPayload) bool {
	if w.dedupe == nil || payload.OutboxID == "" {
		return false
	}
	seen, err := w.dedupe.Seen(ctx, payload.OutboxID)
	if err != nil {
		w.logger.Warn("Email dedupe lookup failed; delivering anyway",
			zap.String("outboxId", payload.OutboxID), zap.Error(err))
		return false
	}
	return seen
}

// rememberDelivered records the delivery AFTER SMTP accepted the message. Marking
// before the send would make the retry path skip every message it was asked to
// retry.
func (w *EmailWorker) rememberDelivered(ctx context.Context, payload dto.EmailEventPayload) {
	if w.dedupe == nil || payload.OutboxID == "" {
		return
	}
	if err := w.dedupe.Mark(ctx, payload.OutboxID, emailDeliveryDedupeTTL); err != nil {
		w.logger.Warn("Email was delivered but the dedupe key could not be written",
			zap.String("outboxId", payload.OutboxID), zap.Error(err))
	}
}

func (w *EmailWorker) Stop() {
	w.BaseWorker.Stop()
}
