package usecase

import (
	"context"
	"slices"
	branderrors "smart-wardrobe-be/internal/modules/brand/application/errors"
	"smart-wardrobe-be/internal/modules/brand/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandmemberrole"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandmemberstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandstatus"
	"smart-wardrobe-be/internal/shared/domain/entities"

	"github.com/google/uuid"
)

// requireBrandRoleShared checks if the given user has the required active member roles for the brand.
// It optimizes database calls by querying the member repository once, leveraging GORM's preloaded Brand relation.
// If the Brand relation is nil (e.g., in unit tests or mock environments), it falls back to querying the brand repository.
func requireBrandRoleShared(
	ctx context.Context,
	brandRepo repositories.IBrandRepository,
	memberRepo repositories.IBrandMemberRepository,
	userID uuid.UUID,
	brandID uuid.UUID,
	allowedRoles ...brandmemberrole.BrandMemberRole,
) error {
	member, err := memberRepo.GetByBrandAndUser(ctx, brandID, userID)
	if err != nil {
		return err
	}
	if member == nil || member.Status != brandmemberstatus.Active {
		return branderrors.ErrBrandPortalForbidden()
	}

	var brand *entities.Brand
	if member.Brand != nil {
		brand = member.Brand
	} else if brandRepo != nil {
		brand, err = brandRepo.GetByID(ctx, brandID)
		if err != nil {
			return err
		}
	}

	if brand == nil {
		return branderrors.ErrBrandNotFound()
	}
	if brand.Status != brandstatus.Active {
		return branderrors.ErrBrandNotActive()
	}

	if slices.Contains(allowedRoles, member.Role) {
		return nil
	}
	return branderrors.ErrBrandPortalForbidden()
}
