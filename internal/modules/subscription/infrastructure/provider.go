package infrastructure

import (
	"smart-wardrobe-be/internal/modules/subscription/infrastructure/persistence"

	"github.com/google/wire"
)

// ProviderSet bundles persistence repository initializers for the subscription module
var ProviderSet = wire.NewSet(
	persistence.NewSubscriptionPlanRepository,
	persistence.NewUserSubscriptionRepository,
	persistence.NewUserDailyQuotaRepository,
)
