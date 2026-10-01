package contract

import (
	"context"

	subscriptionerrors "smart-wardrobe-be/services/account/internal/modules/subscription/application/errors"
	uc_interfaces "smart-wardrobe-be/services/account/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/shared/infrastructure/db"

	"github.com/google/uuid"
)

// IUserSubscriptionProvisioningContract is the narrow, intra-service view the
// identity module needs when it creates an account (024 T196 / FR-013).
//
// It is deliberately NOT a method on `IUserSubscriptionContract`. That interface
// is the cross-service one: `contracts/subscription/v1/client/SubscriptionClient`
// asserts it implements it, and the commerce and AI services bind it over gRPC.
// Provisioning is a WRITE that only ever runs inside the Account service, in the
// caller's transaction, immediately after the user row was inserted — a
// synchronous gRPC hop could not join that transaction, so putting it there would
// have meant shipping a remote call that cannot do its job.
//
// This is the same split 024 T136 made for `AdminGrantLookup`: the gRPC contract
// stays narrow, and the in-process caller gets a separate interface.
//
// The one rule a caller MUST honour: invoke it inside the unit of work that
// created the user row. `user_subscriptions.user_id` is a foreign key onto
// `identity.users(id)`, so provisioning from another connection would run
// against a user row that does not exist there yet.
//
// 024 T202: the rule is ENFORCED here rather than documented. Every write the
// provisioning body performs takes its handle from the context, so a context with
// no transaction sends all three of them to the base connection - each one its own
// implicit transaction. That is precisely the defect 024 T126 fixed on the lazy
// door, and it fails silently: the account ends up holding a plan whose history
// row and outbox record may or may not have landed, and nothing reports it.
//
// This adapter is the right place for it, and it is the only place that covers all
// three account-creation doors. The dependency graph hands the same instance to
// `NewRegisterUseCase`, `NewGoogleAuthUseCase` and `NewUserUseCase`
// (`internal/di/wire_gen.go`), and every one of them passes its transaction
// context straight through. Putting the check in `SubscriptionStateSupport`
// instead would cover strictly less: the use case is reachable directly, and a
// business-logic guard there would break every caller that is not an adapter.
type IUserSubscriptionProvisioningContract interface {
	// EnsureDefaultSubscription provisions the default free plan plus its history
	// row and outbox record, on the caller's open transaction. It is a no-op for
	// an account that already has a subscription.
	EnsureDefaultSubscription(ctx context.Context, userID uuid.UUID) error
}

type userSubscriptionProvisioningContract struct {
	subUC uc_interfaces.ISubscriptionUseCase
}

// NewUserSubscriptionProvisioningContract serves the intra-service provisioning
// view. Wired explicitly so a missing registration is a Wire error rather than a
// nil contract discovered on the first account creation.
func NewUserSubscriptionProvisioningContract(subUC uc_interfaces.ISubscriptionUseCase) IUserSubscriptionProvisioningContract {
	return &userSubscriptionProvisioningContract{subUC: subUC}
}

func (c *userSubscriptionProvisioningContract) EnsureDefaultSubscription(ctx context.Context, userID uuid.UUID) error {
	// `db.GetTx` is the ONLY transaction detection in this codebase - the same call
	// `db.InjectTx` answers to - so checking anything else would accept a context
	// the repositories then refuse to write on.
	if db.GetTx(ctx) == nil {
		return subscriptionerrors.ErrProvisioningRequiresCallerTransaction()
	}
	return c.subUC.EnsureDefaultSubscription(ctx, userID)
}
