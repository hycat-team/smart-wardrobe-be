package contract

import (
	"context"

	uc_interfaces "smart-wardrobe-be/services/account/internal/modules/subscription/application/interface/usecase"

	"github.com/google/uuid"
)

// IUserSubscriptionPlanFilter is the narrow, intra-service view the identity
// module needs to filter the admin account list by membership plan.
//
// It is deliberately NOT a method on `IUserSubscriptionContract`. That interface
// is the cross-service one: `contracts/subscription/v1/client/SubscriptionClient`
// asserts it implements it and the commerce/AI services bind it over gRPC, so
// every method added there needs an RPC. This lookup only ever runs inside the
// Account service, in the same process as the account table it filters, and it
// answers with an id set rather than a paged list — shipping it to another
// service would mean paginating in two places.
//
// Same split as `IUserSubscriptionProvisioningContract` (024 T196) and
// `AdminGrantLookup` (024 T136): the gRPC contract stays narrow, the in-process
// caller gets its own interface.
type IUserSubscriptionPlanFilter interface {
	// ListUserIDsWithEffectivePlan returns every account whose EFFECTIVE plan
	// (the one the admin user table prints) is planID right now. Expired finite
	// plans count as their fallback plan, or as the default plan when the row
	// carries no fallback.
	ListUserIDsWithEffectivePlan(ctx context.Context, planID uuid.UUID) ([]uuid.UUID, error)
}

type userSubscriptionPlanFilter struct {
	subUC uc_interfaces.ISubscriptionUseCase
}

// NewUserSubscriptionPlanFilter serves the intra-service plan-filter view.
// Wired explicitly so a missing registration is a Wire error at startup rather
// than a nil contract discovered the first time an administrator filters.
func NewUserSubscriptionPlanFilter(subUC uc_interfaces.ISubscriptionUseCase) IUserSubscriptionPlanFilter {
	return &userSubscriptionPlanFilter{subUC: subUC}
}

func (c *userSubscriptionPlanFilter) ListUserIDsWithEffectivePlan(ctx context.Context, planID uuid.UUID) ([]uuid.UUID, error) {
	return c.subUC.ListUserIDsWithEffectivePlan(ctx, planID)
}
