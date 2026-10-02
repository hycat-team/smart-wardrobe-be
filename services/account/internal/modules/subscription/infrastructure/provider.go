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
	// 025 T034: the campaign claim ledger. Read by the grant path, by the catch-up
	// sweep and by the operator look-up, all three through the same interface.
	persistence.NewCampaignClaimRepository,
	// 025 T079a: the sweep health table (FR-059). It is a separate repository
	// because the claim ledger is append-only and this table is one mutable row
	// per campaign; merging them would put a mutable flag into an interface whose
	// premise is that nothing is ever updated.
	persistence.NewCampaignSweepRunRepository,
	caching.NewRedisSubscriptionOverviewCache,
	payos.NewPayOSService,
)
