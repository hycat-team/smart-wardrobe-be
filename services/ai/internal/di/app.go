// Package di is the AI Engine service composition root (Wire).
//
// It provides the cross-service contracts AI consumes (Identity, Subscription
// and Admin from Account) over gRPC, alongside the AI orchestrator and cost
// contracts the service hosts (T042, Constitution II, DAG: AI → Account).
package di

import (
	admincontract "smart-wardrobe-be/services/account/contract/admin"
)

// AIContracts is the set of remote contracts AI Engine consumes. Identity auth
// itself (IAuthMiddleware) is verified locally, not over gRPC (see T047).
//
// 024 T178: the `UserSubscription` contract is GONE. It was dialled at startup, stored,
// and passed to the cost use case, which never called it. The only consumer was a
// dependency that did nothing with it.
//
// Removing it is not just tidiness, and it is not a loss of capability: the AI service
// MUST NOT read the Account service's subscription data (Constitution §I, and FR-037
// removed the projection that used to copy it). A remote subscription contract sitting
// in the DI graph is an open door to exactly the cross-domain read this feature
// exists to close, so the door is shut rather than left ajar with a comment.
type AIContracts struct {
	AuditLog admincontract.IAuditLogContract
}
