package application

import (
	uc_interfaces "smart-wardrobe-be/internal/modules/identity/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/identity/application/usecase"
	"smart-wardrobe-be/internal/modules/identity/contract"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	usecase.NewUserUseCase,
	wire.Bind(new(contract.IUserContract), new(uc_interfaces.IUserUseCase)),

	usecase.NewRegisterUseCase,
	usecase.NewSessionUseCase,
	usecase.NewPasswordRecoveryUseCase,
)
