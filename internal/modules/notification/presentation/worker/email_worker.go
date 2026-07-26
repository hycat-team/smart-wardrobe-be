package worker

import (
	"context"
	"encoding/json"

	"smart-wardrobe-be/internal/modules/notification/infrastructure/communication"
	"smart-wardrobe-be/internal/pkg/logger"
	shared_msg "smart-wardrobe-be/internal/shared/infrastructure/messaging"
	"smart-wardrobe-be/internal/shared/observability/workerlog"
	"smart-wardrobe-be/internal/shared/presentation/worker"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

type EmailWorker struct {
	*worker.BaseWorker
	messageClient shared_msg.IRabbitMQClient
	smtpSender    *communication.SmtpSender
	logger        logger.Interface
}

func NewEmailWorker(
	messageClient shared_msg.IRabbitMQClient,
	smtpSender *communication.SmtpSender,
	l logger.Interface,
) *EmailWorker {
	return &EmailWorker{
		BaseWorker:    worker.NewBaseWorker("email"),
		messageClient: messageClient,
		smtpSender:    smtpSender,
		logger:        l,
	}
}

func (w *EmailWorker) Start(ctx context.Context) {
	w.BaseWorker.Start(ctx)
	go w.startConsume(ctx)
}

func (w *EmailWorker) startConsume(ctx context.Context) {
	handler := func(d amqp.Delivery) error {
		var payload struct {
			To      string `json:"to"`
			Subject string `json:"subject"`
			Body    string `json:"body"`
			Channel string `json:"channel"`
		}
		if err := json.Unmarshal(d.Body, &payload); err != nil {
			w.logger.Error("[EmailWorker] Failed to unmarshal message", zap.Error(err))
			return err
		}

		run := workerlog.New("email", workerlog.TriggerQueue)
		run.LogReceived(w.logger,
			zap.String("to", payload.To),
			zap.String("subject", payload.Subject),
		)

		if err := w.smtpSender.Send(ctx, payload.To, payload.Subject, payload.Body); err != nil {
			run.LogFailure(w.logger, err,
				zap.String("to", payload.To),
				zap.String("subject", payload.Subject),
			)
			return err
		}

		run.AddSuccess(1)
		run.LogSuccessIfData(w.logger,
			zap.String("to", payload.To),
			zap.String("subject", payload.Subject),
		)
		return nil
	}

	w.messageClient.RegisterConsumer(shared_msg.QueueNotificationEmail, handler)
	w.logger.Info("[EmailWorker] Registered consumer for queue", zap.String("queue", shared_msg.QueueNotificationEmail))
}

func (w *EmailWorker) Stop() {
	w.BaseWorker.Stop()
}
