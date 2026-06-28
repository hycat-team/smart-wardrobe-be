package benefittype

type BenefitType string

const (
	Voucher       BenefitType = "VOUCHER"
	Discount      BenefitType = "DISCOUNT"
	Gift          BenefitType = "GIFT"
	FreeShipping  BenefitType = "FREE_SHIPPING"
	EarlyAccess   BenefitType = "EARLY_ACCESS"
	FeatureAccess BenefitType = "FEATURE_ACCESS"
)
