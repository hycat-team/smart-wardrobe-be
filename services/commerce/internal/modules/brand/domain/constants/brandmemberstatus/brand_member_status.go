package brandmemberstatus

type BrandMemberStatus string

const (
	Active   BrandMemberStatus = "active"
	Invited  BrandMemberStatus = "invited"
	Disabled BrandMemberStatus = "disabled"
)
