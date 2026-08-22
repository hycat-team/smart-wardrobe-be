package worker

import (
	"context"
	"encoding/json"

	"smart-wardrobe-be/internal/modules/admin/application/dto"
	uc_interfaces "smart-wardrobe-be/internal/modules/admin/application/interface/usecase"
	"smart-wardrobe-be/internal/pkg/logger"
	"smart-wardrobe-be/internal/shared/application/event"
	shared_msg "smart-wardrobe-be/internal/shared/infrastructure/messaging"
	"smart-wardrobe-be/internal/shared/observability/workerlog"
	"smart-wardrobe-be/internal/shared/presentation/worker"

	"go.uber.org/zap"
)

type AuditLogConsumerWorker struct {
	*worker.BaseWorker
	messageClient event.IMessageConsumer
	auditLogUC    uc_interfaces.IAuditLogUseCase
	logger        logger.Interface
}

func NewAuditLogConsumerWorker(
	messageClient event.IMessageConsumer,
	auditLogUC uc_interfaces.IAuditLogUseCase,
	l logger.Interface,
) *AuditLogConsumerWorker {
	return &AuditLogConsumerWorker{
		BaseWorker:    worker.NewBaseWorker("admin_audit_consumer"),
		messageClient: messageClient,
		auditLogUC:    auditLogUC,
		logger:        l,
	}
}

func (w *AuditLogConsumerWorker) Start(ctx context.Context) {
	w.BaseWorker.Start(ctx)
	go w.startConsume(ctx)
}

func (w *AuditLogConsumerWorker) startConsume(ctx context.Context) {
	handler := func(msg event.Message) error {
		var input dto.RecordAuditLogInput
		if err := json.Unmarshal(msg.Body, &input); err != nil {
			w.logger.Error("[AuditLogConsumerWorker] Failed to unmarshal message", zap.Error(err))
			return err
		}

		run := workerlog.New("admin_audit_consumer", workerlog.TriggerQueue)
		run.LogReceived(w.logger,
			zap.String("action", input.Action),
			zap.String("target_type", input.TargetType),
		)

		if err := w.auditLogUC.Record(ctx, input); err != nil {
			run.LogFailure(w.logger, err,
				zap.String("action", input.Action),
				zap.String("target_type", input.TargetType),
			)
			return err
		}

		run.AddTotal(1)
		run.AddSuccess(1)
		run.LogSuccessIfData(w.logger,
			zap.String("action", input.Action),
			zap.String("target_type", input.TargetType),
		)
		return nil
	}

	w.messageClient.RegisterConsumer(shared_msg.QueueAuditLog, handler)
	w.logger.Info("[AuditLogConsumerWorker] Registered consumer for queue", zap.String("queue", shared_msg.QueueAuditLog))
}

func (w *AuditLogConsumerWorker) Stop() {
	w.BaseWorker.Stop()
}
