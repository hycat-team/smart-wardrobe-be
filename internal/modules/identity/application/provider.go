package application

import (
	"smart-wardrobe-be/internal/modules/identity/application/contract"
	"smart-wardrobe-be/internal/modules/identity/application/usecase"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	usecase.NewUserUseCase,
	usecase.NewAuthUseCase,
	contract.NewIdentityModuleContractImpl,
)
