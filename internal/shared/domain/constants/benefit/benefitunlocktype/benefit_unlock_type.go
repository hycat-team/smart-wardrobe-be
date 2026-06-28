package benefitunlocktype

type BenefitUnlockType string

const (
	TierPrivilege   BenefitUnlockType = "TIER_PRIVILEGE"
	PointRedemption BenefitUnlockType = "POINT_REDEMPTION"
	ManualGrant     BenefitUnlockType = "MANUAL_GRANT"
)
