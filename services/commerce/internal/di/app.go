// Package di is the Commerce Core service composition root (Wire).
//
// It provides the cross-service contracts Commerce consumes (Identity,
// Subscription, Admin from Account; AIOrchestrator and cost readers from AI)
// over gRPC. Commerce hosts no gRPC service (T041, Constitution II, DAG).
package di

import (
	admincontract "smart-wardrobe-be/services/account/contract/admin"
	identitycontract "smart-wardrobe-be/services/account/contract/identity"
	subscriptioncontract "smart-wardrobe-be/services/account/contract/subscription"
	aicontract "smart-wardrobe-be/services/ai/contract/aiexecutor"
)

// CommerceContracts is the set of remote contracts Commerce Core consumes.
type CommerceContracts struct {
	UserBrief             identitycontract.IUserBriefContract
	UserDashboardReader   identitycontract.IUserDashboardReader
	UserSubscription      subscriptioncontract.IUserSubscriptionContract
	UserQuota             subscriptioncontract.IUserQuotaContract
	SubscriptionDashboard subscriptioncontract.ISubscriptionDashboardReader
	AuditLog              admincontract.IAuditLogContract
	AIOrchestrator        aicontract.IAIOrchestrator
	AICostDashboardReader aicontract.IAICostDashboardReader
}
