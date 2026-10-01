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
	// CampaignGrant consumes account-creation messages and grants the campaign
	// plan (025 T056). Placed after the webhook workers because it is a
	// queue consumer like they are, and before the notification worker because a
	// grant enqueues the email it will later deliver — starting them in this order
	// means the email worker is already consuming by the time the first grant
	// arrives.
	CampaignGrant    *subscriptionworker.CampaignGrantWorker
	Email            *notificationworker.EmailWorker
	AuditLogConsumer *adminworker.AuditLogConsumerWorker
	AuditCleanup     *adminworker.AuditCleanupWorker
	Outbox           *sharedWorker.OutboxWorker
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
		w.CampaignGrant,
		w.Email,
		w.AuditLogConsumer,
		w.AuditCleanup,
		w.Outbox,
	}
}
