package brandmemberrole

type BrandMemberRole string

const (
	Owner        BrandMemberRole = "owner"
	Manager      BrandMemberRole = "manager"
	SupportStaff BrandMemberRole = "support_staff"
	Marketer     BrandMemberRole = "marketer"
)
