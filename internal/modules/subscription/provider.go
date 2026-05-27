package subscription

import (
	"smart-wardrobe-be/internal/modules/subscription/application/contract"
	"smart-wardrobe-be/internal/modules/subscription/infrastructure/persistence"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	persistence.NewSubscriptionPlanRepository,
	contract.NewSubscriptionModuleContractImpl,
)
