package deposittransactiontype

type DepositTransactionType string

const (
	DirectPurchase DepositTransactionType = "DIRECT_PURCHASE"
	WalletTopup    DepositTransactionType = "WALLET_TOPUP"
)
