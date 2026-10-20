package campaign

import (
	"context"

	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
)

// noopPlanPeriodRecorder keeps the campaign harness compiling after 031 added
// the recorder dependency to PlanChangePublisher. The campaign tests assert the
// event path; interval behaviour is tested with the recorder itself.
type noopPlanPeriodRecorder struct{}

func (noopPlanPeriodRecorder) Record(
	context.Context,
	*entities.UserSubscriptionEvent,
	*entities.UserSubscription,
	*entities.SubscriptionPlan,
) error {
	return nil
}
