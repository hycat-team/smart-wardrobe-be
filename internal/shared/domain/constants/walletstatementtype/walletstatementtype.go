package walletstatementtype

type WalletStatementType string

const (
	Topup                WalletStatementType = "TOPUP"
	SubscriptionPurchase WalletStatementType = "SUBSCRIPTION_PURCHASE"
	SubscriptionRenewal  WalletStatementType = "SUBSCRIPTION_RENEWAL"
)
