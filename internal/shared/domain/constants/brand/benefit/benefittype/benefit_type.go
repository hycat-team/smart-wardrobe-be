package benefittype

type BenefitType string

const (
	Voucher       BenefitType = "voucher"
	Discount      BenefitType = "discount"
	Gift          BenefitType = "gift"
	FreeShipping  BenefitType = "free_shipping"
	EarlyAccess   BenefitType = "early_access"
	FeatureAccess BenefitType = "feature_access"
)
