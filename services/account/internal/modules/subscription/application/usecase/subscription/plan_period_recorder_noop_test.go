package subscription

import (
	"context"

	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
)

// noopPlanPeriodRecorder keeps the pre-031 publisher tests compiling. The
// recorder's own behaviour is covered by the feature-031 tests; these call
// sites exercise the event path, not the interval table.
type noopPlanPeriodRecorder struct{}

func (noopPlanPeriodRecorder) Record(
	context.Context,
	*entities.UserSubscriptionEvent,
	*entities.UserSubscription,
	*entities.SubscriptionPlan,
) error {
	return nil
}
