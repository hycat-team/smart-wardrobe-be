package loyaltytransactiontype

type LoyaltyTransactionType string

const (
	Earn   LoyaltyTransactionType = "EARN"
	Redeem LoyaltyTransactionType = "REDEEM"
	Adjust LoyaltyTransactionType = "ADJUST"
	Expire LoyaltyTransactionType = "EXPIRE"
	Refund LoyaltyTransactionType = "REFUND"
)
