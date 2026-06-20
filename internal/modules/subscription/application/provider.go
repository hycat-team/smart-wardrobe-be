package application

import (
	uc_interfaces "smart-wardrobe-be/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/subscription/application/usecase/plan"
	"smart-wardrobe-be/internal/modules/subscription/application/usecase/purchase"
	"smart-wardrobe-be/internal/modules/subscription/application/usecase/quota"
	"smart-wardrobe-be/internal/modules/subscription/application/usecase/resolution"
	"smart-wardrobe-be/internal/modules/subscription/application/usecase/subscription"
	"smart-wardrobe-be/internal/modules/subscription/application/usecase/wallet"
	"smart-wardrobe-be/internal/modules/subscription/application/usecase/webhook"
	"smart-wardrobe-be/internal/modules/subscription/application/validator"
	"smart-wardrobe-be/internal/modules/subscription/contract"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	subscription.NewSubscriptionUseCase,
	wire.Bind(new(contract.IUserSubscriptionContract), new(uc_interfaces.ISubscriptionUseCase)),

	plan.NewSubscriptionPlanUseCase,
	wire.Bind(new(contract.ISubscriptionPlanContract), new(uc_interfaces.ISubscriptionPlanUseCase)),

	quota.NewUserQuotaUseCase,
	wire.Bind(new(contract.IUserQuotaContract), new(uc_interfaces.IUserQuotaUseCase)),

	wallet.NewWalletUseCase,
	purchase.NewSubscriptionPurchaseUseCase,
	webhook.NewPaymentWebhookUseCase,
	resolution.NewPaymentResolutionUseCase,
	validator.NewSubscriptionCatalogValidator,
)
