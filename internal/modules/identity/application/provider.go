package application

import (
	"smart-wardrobe-be/internal/modules/identity/application/usecase"
	"smart-wardrobe-be/internal/modules/identity/presentation/contract"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	usecase.NewUserUseCase,
	contract.NewUserContract,

	usecase.NewRegisterUseCase,
	usecase.NewSessionUseCase,
	usecase.NewPasswordRecoveryUseCase,
)
