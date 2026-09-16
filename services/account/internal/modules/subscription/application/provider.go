package application

import (
	"smart-wardrobe-be/services/account/contract/subscription"
	admin_billing "smart-wardrobe-be/services/account/internal/modules/subscription/application/usecase/admin_billing"
	"smart-wardrobe-be/services/account/internal/modules/subscription/application/usecase/admin_stats"
	"smart-wardrobe-be/services/account/internal/modules/subscription/application/usecase/plan"
	"smart-wardrobe-be/services/account/internal/modules/subscription/application/usecase/purchase"
	"smart-wardrobe-be/services/account/internal/modules/subscription/application/usecase/quota"
	"smart-wardrobe-be/services/account/internal/modules/subscription/application/usecase/reconciliation"
	"smart-wardrobe-be/services/account/internal/modules/subscription/application/usecase/resolution"
	"smart-wardrobe-be/services/account/internal/modules/subscription/application/usecase/subscription"
	"smart-wardrobe-be/services/account/internal/modules/subscription/application/usecase/wallet"
	"smart-wardrobe-be/services/account/internal/modules/subscription/application/usecase/webhook"
	"smart-wardrobe-be/services/account/internal/modules/subscription/application/usecase/webhook_inbox"
	"smart-wardrobe-be/services/account/internal/modules/subscription/application/validator"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	subscription.NewSubscriptionUseCase,
	contract.NewUserSubscriptionContract,

	plan.NewSubscriptionPlanUseCase,

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
