package benefitredemptionstatus

type BenefitRedemptionStatus string

const (
	Pending   BenefitRedemptionStatus = "pending"
	Redeemed  BenefitRedemptionStatus = "redeemed"
	Used      BenefitRedemptionStatus = "used"
	Cancelled BenefitRedemptionStatus = "cancelled"
	Expired   BenefitRedemptionStatus = "expired"
)
