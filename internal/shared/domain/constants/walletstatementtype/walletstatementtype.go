package walletstatementtype

type WalletStatementType string

const (
	Topup                     WalletStatementType = "TOPUP"
	SubscriptionPurchase      WalletStatementType = "SUBSCRIPTION_PURCHASE"
	SubscriptionRenewal       WalletStatementType = "SUBSCRIPTION_RENEWAL"
	LowerTierPaymentCredit    WalletStatementType = "LOWER_TIER_PAYMENT_CREDIT"
	SameLifetimePaymentCredit WalletStatementType = "SAME_LIFETIME_PAYMENT_CREDIT"
)
