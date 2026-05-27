package application

import (
	"smart-wardrobe-be/internal/modules/subscription/application/contract"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	contract.NewSubscriptionModuleContractImpl,
)
