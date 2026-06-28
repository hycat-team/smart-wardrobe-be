package brandmemberstatus

type BrandMemberStatus string

const (
	Active   BrandMemberStatus = "ACTIVE"
	Invited  BrandMemberStatus = "INVITED"
	Disabled BrandMemberStatus = "DISABLED"
)
