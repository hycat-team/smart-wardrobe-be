package application

import (
	"smart-wardrobe-be/services/account/contract/subscription"
	admin_billing "smart-wardrobe-be/services/account/internal/modules/subscription/application/usecase/admin_billing"
	"smart-wardrobe-be/services/account/internal/modules/subscription/application/usecase/admin_stats"
	"smart-wardrobe-be/services/account/internal/modules/subscription/application/usecase/campaign"
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

	// 025 T055: the campaign grant use case and the quota rules it depends on.
	// The constructor is wired here rather than where the worker is registered so
	// that a missing dependency is a DI error at startup, not a nil use case
	// discovered by the first account that registers after a campaign opens.
	campaign.NewCampaignUseCase,

	// 025 T079: the read-only operator look-up. It is a SEPARATE use case from
	// `campaign.NewCampaignUseCase` on purpose: the grant path writes and holds a
	// unit of work, this one must not, and sharing a type would put both in the
	// same object where FR-069 would survive only as a habit. Wiring it here also
	// means a missing dependency is a DI error at startup rather than a nil
	// pointer on the first support request.
	campaign.NewCampaignQueryUseCase,

	// 025 T097: the catch-up sweep. It depends on `ICampaignUseCase` rather than
	// on the grant use case's dependencies, so registering it HERE is what forces
	// the sweep to reach the rules through the shared entry point. A sweep wired
	// to the repositories directly would be free to grow its own eligibility
	// check, and FR-080's whole protection is that it cannot.
	campaign.NewCampaignSweepUseCase,

	// 026 T035: the campaign registry WRITER. It is a third campaign use case, not a
	// second method on either of the two above, for the reason
	// `ICampaignRegistryUseCase` gives in full — the grant path, the look-up and the
	// operator path have three different shapes of input, three different transactions
	// and three different blast radii.
	//
	// Wired here so that a missing dependency is a DI error at startup rather than a nil
	// use case discovered by the first person who tries to open a campaign on a launch
	// night — which is precisely the moment the feature was built for.
	campaign.NewCampaignRegistryUseCase,
)
