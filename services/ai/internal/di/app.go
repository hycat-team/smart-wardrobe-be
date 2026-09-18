// Package di is the AI Engine service composition root (Wire).
//
// It provides the cross-service contracts AI consumes (Identity, Subscription
// and Admin from Account) over gRPC, alongside the AI orchestrator and cost
// contracts the service hosts (T042, Constitution II, DAG: AI → Account).
package di

import (
	admincontract "smart-wardrobe-be/services/account/contract/admin"
	subscriptioncontract "smart-wardrobe-be/services/account/contract/subscription"
)

// AIContracts is the set of remote contracts AI Engine consumes. Identity auth
// itself (IAuthMiddleware) is verified locally, not over gRPC (see T047).
type AIContracts struct {
	UserSubscription subscriptioncontract.IUserSubscriptionContract
	AuditLog         admincontract.IAuditLogContract
}
