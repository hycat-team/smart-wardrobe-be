package contract

import (
	"context"

	"github.com/google/uuid"
)

type IBillingModuleContract interface {
	GetDefaultSubscriptionPlanID(ctx context.Context) (uuid.UUID, error)
	IsPremiumPlan(ctx context.Context, planID uuid.UUID) (bool, error)
}
