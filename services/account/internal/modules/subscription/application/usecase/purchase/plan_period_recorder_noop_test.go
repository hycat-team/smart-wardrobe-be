package purchase

import (
	"context"

	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
)

// noopPlanPeriodRecorder keeps the purchase harness compiling after 031 added
// the recorder dependency to PlanChangePublisher.
type noopPlanPeriodRecorder struct{}

func (noopPlanPeriodRecorder) Record(
	context.Context,
	*entities.UserSubscriptionEvent,
	*entities.UserSubscription,
	*entities.SubscriptionPlan,
) error {
	return nil
}
