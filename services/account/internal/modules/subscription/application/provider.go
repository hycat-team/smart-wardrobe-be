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
	// 024 T196: the narrow intra-service view the identity module calls right
	// after inserting a user row. Separate from the cross-service contract above,
	// which is served over gRPC and cannot join the caller's transaction.
	contract.NewUserSubscriptionProvisioningContract,

	// 024 T045: the single publish point for subscription.plan.changed. Every
	// path that changes a plan depends on it, so "forgot to publish" becomes a
	// wiring error rather than a silent production gap.
	subscription.NewPlanChangePublisher,

	// 024 T080: resolves an administrator-supplied identifier (UUID, username or
	// email) to an account. Wired explicitly because the grant use case depends
	// on it directly rather than through the identity contract interface.
	subscription.NewAdminUserResolver,

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
