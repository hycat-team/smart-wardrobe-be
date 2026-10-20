package infrastructure

import (
	"smart-wardrobe-be/services/account/internal/modules/subscription/infrastructure/caching"
	"smart-wardrobe-be/services/account/internal/modules/subscription/infrastructure/payment/payos"
	"smart-wardrobe-be/services/account/internal/modules/subscription/infrastructure/persistence"

	"github.com/google/wire"
)

// ProviderSet bundles persistence repository initializers for the subscription module
var ProviderSet = wire.NewSet(
	persistence.NewSubscriptionPlanRepository,
	persistence.NewUserSubscriptionRepository,
	persistence.NewUserDailyQuotaRepository,
	persistence.NewUserWalletRepository,
	persistence.NewDepositTransactionRepository,
	persistence.NewWalletStatementRepository,
	persistence.NewPaymentEventRepository,
	persistence.NewWebhookInboxRepository,
	persistence.NewUserSubscriptionEventRepository,
	persistence.NewSubscriptionRenewalAttemptRepository,
	// 031 T008: the validity-interval history read/written by the recorder and
	// by the distribution report.
	persistence.NewUserPlanPeriodRepository,
	// 025 T034: the campaign claim ledger. Read by the grant path, by the catch-up
	// sweep and by the operator look-up, all three through the same interface.
	persistence.NewCampaignClaimRepository,
	// 025 T079a: the sweep health table (FR-059). It is a separate repository
	// because the claim ledger is append-only and this table is one mutable row
	// per campaign; merging them would put a mutable flag into an interface whose
	// premise is that nothing is ever updated.
	persistence.NewCampaignSweepRunRepository,
	// 026 T020: the campaign REGISTRY (FR-001). Registered before the claim ledger
	// because it is now the source the ledger's own paths ask "which campaigns
	// exist" — and registering it here is what makes a missing registry a wiring
	// error at startup rather than a nil interface discovered by the first account
	// created after a campaign opens.
	//
	// It has no delete path at all (FR-005), so nothing else in this set can
	// remove a campaign row out from under the ledger that references it by text.
	persistence.NewCampaignRepository,
	caching.NewRedisSubscriptionOverviewCache,
	payos.NewPayOSService,
)
