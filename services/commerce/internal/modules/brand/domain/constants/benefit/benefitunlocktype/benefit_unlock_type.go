package benefitunlocktype

type BenefitUnlockType string

const (
	TierPrivilege   BenefitUnlockType = "tier_privilege"
	PointRedemption BenefitUnlockType = "point_redemption"
	ManualGrant     BenefitUnlockType = "manual_grant"
)
