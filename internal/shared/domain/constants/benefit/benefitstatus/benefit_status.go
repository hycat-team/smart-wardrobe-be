package benefitstatus

type BenefitStatus string

const (
	Active   BenefitStatus = "ACTIVE"
	Inactive BenefitStatus = "INACTIVE"
	Archived BenefitStatus = "ARCHIVED"
)
