// Package di is the Account service composition root (Wire).
//
// It composes the standalone Account service: HTTP routers, gRPC servers
// (Identity, Subscription, Admin), background workers and in-process contract
// implementations. The service hosts gRPC and makes no outbound calls
// (T040, Constitution II).
package di

import (
	admincontract "smart-wardrobe-be/services/account/contract/admin"
	identitycontract "smart-wardrobe-be/services/account/contract/identity"
	subscriptioncontract "smart-wardrobe-be/services/account/contract/subscription"
	admingrpc "smart-wardrobe-be/services/account/internal/modules/admin/presentation/grpc"
	adminrouter "smart-wardrobe-be/services/account/internal/modules/admin/presentation/router"
	adminworker "smart-wardrobe-be/services/account/internal/modules/admin/presentation/worker"
	identitygrpc "smart-wardrobe-be/services/account/internal/modules/identity/presentation/grpc"
	identityrouter "smart-wardrobe-be/services/account/internal/modules/identity/presentation/router"
	notificationworker "smart-wardrobe-be/services/account/internal/modules/notification/presentation/worker"
	subscriptiongrpc "smart-wardrobe-be/services/account/internal/modules/subscription/presentation/grpc"
	subscriptionrouter "smart-wardrobe-be/services/account/internal/modules/subscription/presentation/router"
	subscriptionworker "smart-wardrobe-be/services/account/internal/modules/subscription/presentation/worker"
	sharedbootstrap "smart-wardrobe-be/shared/bootstrap"
)

// AccountApp is the composed standalone Account service (T040).
type AccountApp struct {
	IdentityRouter     *identityrouter.Router
	SubscriptionRouter *subscriptionrouter.Router
	AdminRouter        *adminrouter.Router

	SubscriptionRenewalWorker   *subscriptionworker.SubscriptionRenewalWorker
	PaymentReconciliationWorker *subscriptionworker.PaymentReconciliationWorker
	WebhookInboxWorker          *subscriptionworker.WebhookInboxWorker
	EmailWorker                 *notificationworker.EmailWorker
	AuditLogConsumerWorker      *adminworker.AuditLogConsumerWorker
	AuditCleanupWorker          *adminworker.AuditCleanupWorker

	UserBriefContract           identitycontract.IUserBriefContract
	UserDashboardReader         identitycontract.IUserDashboardReader
	AuthMiddleware              identitycontract.IAuthMiddleware
	UserSubscriptionContract    subscriptioncontract.IUserSubscriptionContract
	UserQuotaContract           subscriptioncontract.IUserQuotaContract
	SubscriptionDashboardReader subscriptioncontract.ISubscriptionDashboardReader
	AuditLogContract            admincontract.IAuditLogContract
	// Validator runs catalog checks before boot (wired into the app lifecycle).
	Validator sharedbootstrap.StartupValidator

	IdentityServer              *identitygrpc.IdentityServer
	IdentityDashboardServer     *identitygrpc.IdentityDashboardServer
	SubscriptionServer          *subscriptiongrpc.SubscriptionServer
	SubscriptionDashboardServer *subscriptiongrpc.SubscriptionDashboardServer
	AdminServer                 *admingrpc.AdminServer
}
