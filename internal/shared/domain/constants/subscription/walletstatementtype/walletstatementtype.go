package walletstatementtype

type WalletStatementType string

const (
	Topup                     WalletStatementType = "topup"
	SubscriptionPurchase      WalletStatementType = "subscription_purchase"
	SubscriptionRenewal       WalletStatementType = "subscription_renewal"
	LowerTierPaymentCredit    WalletStatementType = "lower_tier_payment_credit"
	SameLifetimePaymentCredit WalletStatementType = "same_lifetime_payment_credit"
)
