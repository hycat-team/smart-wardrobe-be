package usecase

import (
	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandmemberrole"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandstatus"
)

func isValidBrandStatus(status brandstatus.BrandStatus) bool {
	switch status {
	case brandstatus.PendingReview, brandstatus.Active, brandstatus.Suspended, brandstatus.Archived:
		return true
	default:
		return false
	}
}

func isValidBrandMemberRole(role brandmemberrole.BrandMemberRole) bool {
	switch role {
	case brandmemberrole.Owner, brandmemberrole.Staff:
		return true
	default:
		return false
	}
}
