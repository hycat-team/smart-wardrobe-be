package benefitstatus

type BenefitStatus string

const (
	Active   BenefitStatus = "active"
	Inactive BenefitStatus = "inactive"
	Archived BenefitStatus = "archived"
)
