package application

import (
	admin_billing "smart-wardrobe-be/internal/modules/subscription/application/usecase/admin_billing"
	"smart-wardrobe-be/internal/modules/subscription/application/usecase/admin_stats"
	"smart-wardrobe-be/internal/modules/subscription/application/usecase/plan"
	"smart-wardrobe-be/internal/modules/subscription/application/usecase/purchase"
	"smart-wardrobe-be/internal/modules/subscription/application/usecase/quota"
	"smart-wardrobe-be/internal/modules/subscription/application/usecase/reconciliation"
	"smart-wardrobe-be/internal/modules/subscription/application/usecase/resolution"
	"smart-wardrobe-be/internal/modules/subscription/application/usecase/subscription"
	"smart-wardrobe-be/internal/modules/subscription/application/usecase/wallet"
	"smart-wardrobe-be/internal/modules/subscription/application/usecase/webhook"
	"smart-wardrobe-be/internal/modules/subscription/application/usecase/webhook_inbox"
	"smart-wardrobe-be/internal/modules/subscription/application/validator"
	"smart-wardrobe-be/internal/modules/subscription/presentation/contract"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	subscription.NewSubscriptionUseCase,
	contract.NewUserSubscriptionContract,

	plan.NewSubscriptionPlanUseCase,
	contract.NewSubscriptionPlanContract,

	quota.NewUserQuotaUseCase,
	contract.NewUserQuotaContract,

	admin_stats.NewAdminStatsUseCase,
	contract.NewSubscriptionDashboardReader,

	wallet.NewWalletUseCase,
	purchase.NewSubscriptionPurchaseUseCase,
	webhook.NewPaymentWebhookUseCase,
	resolution.NewPaymentResolutionUseCase,
	admin_billing.NewAdminBillingUseCase,
	reconciliation.NewPaymentReconciliationUseCase,
	webhook_inbox.NewWebhookInboxUseCase,
	validator.NewSubscriptionCatalogValidator,
)
