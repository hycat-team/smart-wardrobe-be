package benefitredemptionstatus

type BenefitRedemptionStatus string

const (
	Pending   BenefitRedemptionStatus = "PENDING"
	Redeemed  BenefitRedemptionStatus = "REDEEMED"
	Used      BenefitRedemptionStatus = "USED"
	Cancelled BenefitRedemptionStatus = "CANCELLED"
	Expired   BenefitRedemptionStatus = "EXPIRED"
)
