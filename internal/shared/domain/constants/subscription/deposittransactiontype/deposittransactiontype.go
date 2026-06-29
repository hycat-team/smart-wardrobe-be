package deposittransactiontype

type DepositTransactionType string

const (
	DirectPurchase DepositTransactionType = "direct_purchase"
	WalletTopup    DepositTransactionType = "wallet_topup"
)
