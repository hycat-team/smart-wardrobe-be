package application

import (
	uc_interfaces "smart-wardrobe-be/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/subscription/application/usecase"
	"smart-wardrobe-be/internal/modules/subscription/contract"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	usecase.NewSubscriptionUseCase,
	wire.Bind(new(contract.IUserSubscriptionContract), new(uc_interfaces.ISubscriptionUseCase)),

	usecase.NewSubscriptionPlanUseCase,
	wire.Bind(new(contract.ISubscriptionPlanContract), new(uc_interfaces.ISubscriptionPlanUseCase)),

	usecase.NewUserQuotaUseCase,
	wire.Bind(new(contract.IUserQuotaContract), new(uc_interfaces.IUserQuotaUseCase)),

	usecase.NewWalletUseCase,
	usecase.NewSubscriptionPurchaseUseCase,
	usecase.NewPaymentWebhookUseCase,
)
