package brand

import (
	"context"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/application/usecase"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/constants/brandstatus"
	"smart-wardrobe-be/shared/domain/constants/brandmemberrole"

	"github.com/google/uuid"
)

func (uc *BrandUseCase) RequireBrandRole(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, allowedRoles ...brandmemberrole.BrandMemberRole) error {
	return usecase.RequireBrandRoleShared(ctx, uc.brandRepo, uc.memberRepo, userID, brandID, allowedRoles...)
}

func isValidBrandStatus(status brandstatus.BrandStatus) bool {
	switch status {
	case brandstatus.PendingReview, brandstatus.Active, brandstatus.Suspended, brandstatus.Archived, brandstatus.Rejected:
		return true
	default:
		return false
	}
}
