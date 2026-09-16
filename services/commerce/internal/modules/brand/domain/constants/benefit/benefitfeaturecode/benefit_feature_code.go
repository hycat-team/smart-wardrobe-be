package benefitfeaturecode

import "strings"

type BenefitFeatureCode string

const (
	SampleMixAccess BenefitFeatureCode = "sample_mix_access"
)

func Parse(value string) (BenefitFeatureCode, bool) {
	code := BenefitFeatureCode(strings.ToLower(strings.TrimSpace(value)))
	if code.IsValid() {
		return code, true
	}
	return "", false
}

func (c BenefitFeatureCode) IsValid() bool {
	switch c {
	case SampleMixAccess:
		return true
	default:
		return false
	}
}
