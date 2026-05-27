package billing

import (
	"smart-wardrobe-be/internal/modules/billing/application/contract"
	"smart-wardrobe-be/internal/modules/billing/infrastructure/persistence"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	persistence.NewSubscriptionPlanRepository,
	contract.NewBillingModuleContractImpl,
)
