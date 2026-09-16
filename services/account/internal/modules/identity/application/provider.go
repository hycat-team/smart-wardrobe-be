package application

import (
	"smart-wardrobe-be/services/account/contract/identity"
	"smart-wardrobe-be/services/account/internal/modules/identity/application/usecase"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	usecase.NewUserUseCase,
	usecase.NewUserBriefUseCase,
	contract.NewUserBriefContract,

	usecase.NewRegisterUseCase,
	usecase.NewSessionUseCase,
	usecase.NewPasswordRecoveryUseCase,
)
