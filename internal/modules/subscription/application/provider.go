package application

import (
	"smart-wardrobe-be/internal/modules/subscription/application/contract"
	"smart-wardrobe-be/internal/modules/subscription/application/usecase"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	contract.NewSubscriptionModuleContractImpl,
	usecase.NewSubscriptionUseCase,
)
