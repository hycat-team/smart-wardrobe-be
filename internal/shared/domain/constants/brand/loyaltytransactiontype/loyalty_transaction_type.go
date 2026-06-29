package loyaltytransactiontype

type LoyaltyTransactionType string

const (
	Earn   LoyaltyTransactionType = "earn"
	Redeem LoyaltyTransactionType = "redeem"
	Adjust LoyaltyTransactionType = "adjust"
	Expire LoyaltyTransactionType = "expire"
	Refund LoyaltyTransactionType = "refund"
)
