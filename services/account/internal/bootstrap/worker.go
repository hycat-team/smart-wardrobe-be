package bootstrap

import (
	"context"

	adminworker "smart-wardrobe-be/services/account/internal/modules/admin/presentation/worker"
	notificationworker "smart-wardrobe-be/services/account/internal/modules/notification/presentation/worker"
	subscriptionworker "smart-wardrobe-be/services/account/internal/modules/subscription/presentation/worker"
	sharedWorker "smart-wardrobe-be/shared/presentation/worker"
)

// Worker is the lifecycle contract every background worker satisfies through
// the shared BaseWorker (FR-044/FR-045).
type Worker interface {
	Start(ctx context.Context)
	Stop()
}

// Workers groups the Account service background workers. They run only in
// APP_MODE=worker|all; APP_MODE=http never starts them (FR-044, SC-018).
type Workers struct {
	Renewal               *subscriptionworker.SubscriptionRenewalWorker
	PaymentReconciliation *subscriptionworker.PaymentReconciliationWorker
	WebhookInbox          *subscriptionworker.WebhookInboxWorker
	Email                 *notificationworker.EmailWorker
	AuditLogConsumer      *adminworker.AuditLogConsumerWorker
	AuditCleanup          *adminworker.AuditCleanupWorker
	Outbox                *sharedWorker.OutboxWorker
}

// All returns the workers in a stable start/stop order.
func (w *Workers) All() []Worker {
	if w == nil {
		return nil
	}
	return []Worker{
		w.Renewal,
		w.PaymentReconciliation,
		w.WebhookInbox,
		w.Email,
		w.AuditLogConsumer,
		w.AuditCleanup,
		w.Outbox,
	}
}
