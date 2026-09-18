package worker

import (
	"context"

	"smart-wardrobe-be/services/account/dto/admin"
	uc_interfaces "smart-wardrobe-be/services/account/internal/modules/admin/application/interface/usecase"
	"smart-wardrobe-be/shared/application/constants/eventconstants"
	"smart-wardrobe-be/shared/application/event"
	"smart-wardrobe-be/shared/observability/workerlog"
	"smart-wardrobe-be/shared/pkg/logger"
	"smart-wardrobe-be/shared/presentation/worker"

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
	if err := event.RegisterTypedConsumer(w.messageClient, eventconstants.QueueAuditLog, w.handle, event.ConsumerOpts{}); err != nil {
		w.logger.Error("Failed to register audit log consumer", zap.Error(err))
	}
}

func (w *AuditLogConsumerWorker) handle(ctx context.Context, input dto.RecordAuditLogInput) error {
	run := workerlog.New("admin_audit_consumer", workerlog.TriggerQueue)
	run.LogReceived(w.logger,
		zap.String("correlationId", event.CorrelationIdFromContext(ctx)),
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

func (w *AuditLogConsumerWorker) Stop() {
	w.BaseWorker.Stop()
}
