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
	// 025 T034: the campaign claim ledger. Not consumed by anything yet — it is
	// wired here so the constructor stays reachable once the campaign use cases
	// land, and so its signature is checked at DI time rather than at first use.
	persistence.NewCampaignClaimRepository,
	caching.NewRedisSubscriptionOverviewCache,
	payos.NewPayOSService,
)
