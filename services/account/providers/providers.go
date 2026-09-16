// Package providers exposes the Account service's composition surface for the
// transitional monolith (internal/di, internal/bootstrap, internal/api).
//
// It re-exports the service's wire ProviderSets plus the concrete router and
// worker types the monolith composes. It MUST NOT import implementation
// packages of other services. Removed/replaced by per-service composition
// (T040) when cmd/server leaves main at T034.
package providers

import (
	admincontract "smart-wardrobe-be/services/account/contract/admin"
	identitycontract "smart-wardrobe-be/services/account/contract/identity"
	subscriptioncontract "smart-wardrobe-be/services/account/contract/subscription"
	adminmodule "smart-wardrobe-be/services/account/internal/modules/admin"
	adminrouter "smart-wardrobe-be/services/account/internal/modules/admin/presentation/router"
	adminworker "smart-wardrobe-be/services/account/internal/modules/admin/presentation/worker"
	identitymodule "smart-wardrobe-be/services/account/internal/modules/identity"
	identityrouter "smart-wardrobe-be/services/account/internal/modules/identity/presentation/router"
	notificationmodule "smart-wardrobe-be/services/account/internal/modules/notification"
	notificationworker "smart-wardrobe-be/services/account/internal/modules/notification/presentation/worker"
	subscriptionmodule "smart-wardrobe-be/services/account/internal/modules/subscription"
	subscriptionrouter "smart-wardrobe-be/services/account/internal/modules/subscription/presentation/router"
	subscriptionworker "smart-wardrobe-be/services/account/internal/modules/subscription/presentation/worker"
	sharedbootstrap "smart-wardrobe-be/shared/bootstrap"

	"github.com/google/wire"
)

// ProviderSet aggregates the Account service modules (identity, subscription,
// admin, notification).
var ProviderSet = wire.NewSet(
	identitymodule.ProviderSet,
	notificationmodule.ProviderSet,
	subscriptionmodule.ProviderSet,
	adminmodule.ProviderSet,
)

// Router aliases for monolith composition.
type (
	IdentityRouter     = identityrouter.Router
	SubscriptionRouter = subscriptionrouter.Router
	AdminRouter        = adminrouter.Router
)

// Worker aliases for monolith composition.
type (
	SubscriptionRenewalWorker   = subscriptionworker.SubscriptionRenewalWorker
	PaymentReconciliationWorker = subscriptionworker.PaymentReconciliationWorker
	WebhookInboxWorker          = subscriptionworker.WebhookInboxWorker
	EmailWorker                 = notificationworker.EmailWorker
	AuditLogConsumerWorker      = adminworker.AuditLogConsumerWorker
	AuditCleanupWorker          = adminworker.AuditCleanupWorker
)

// AccountBundle is the composed product of the Account service: routers and
// workers for the monolith engine plus the cross-service contracts it owns
// (consumed in-process by the monolith, via gRPC by deployed services).
type AccountBundle struct {
	IdentityRouter     *IdentityRouter
	SubscriptionRouter *SubscriptionRouter
	AdminRouter        *AdminRouter

	SubscriptionRenewalWorker   *SubscriptionRenewalWorker
	PaymentReconciliationWorker *PaymentReconciliationWorker
	WebhookInboxWorker          *WebhookInboxWorker
	EmailWorker                 *EmailWorker
	AuditLogConsumerWorker      *AuditLogConsumerWorker
	AuditCleanupWorker          *AuditCleanupWorker

	UserBriefContract           identitycontract.IUserBriefContract
	UserDashboardReader         identitycontract.IUserDashboardReader
	AuthMiddleware              identitycontract.IAuthMiddleware
	UserSubscriptionContract    subscriptioncontract.IUserSubscriptionContract
	UserQuotaContract           subscriptioncontract.IUserQuotaContract
	SubscriptionDashboardReader subscriptioncontract.ISubscriptionDashboardReader
	AuditLogContract            admincontract.IAuditLogContract
	// Validator runs catalog checks before boot (wired into bootstrap.App).
	Validator sharedbootstrap.StartupValidator
}
