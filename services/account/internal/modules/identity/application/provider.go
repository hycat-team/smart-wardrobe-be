package application

import (
	"smart-wardrobe-be/services/account/contract/identity"
	"smart-wardrobe-be/services/account/internal/modules/identity/application/usecase"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	usecase.NewUserUseCase,
	usecase.NewUserBriefUseCase,
	// 024 T136: the constructor returns the CONCRETE type so the admin grant
	// flow can be given the narrow `AdminGrantLookup` view (which carries the
	// soft-delete-aware method the full interface deliberately omits). The bind
	// below then hands the interface to every existing consumer, so none of them
	// had to change.
	contract.NewUserBriefContract,
	wire.Bind(new(contract.IUserBriefContract), new(*contract.UserBriefContract)),

	usecase.NewRegisterUseCase,
	usecase.NewSessionUseCase,
	usecase.NewPasswordRecoveryUseCase,
	usecase.NewGoogleAuthUseCase,
)
