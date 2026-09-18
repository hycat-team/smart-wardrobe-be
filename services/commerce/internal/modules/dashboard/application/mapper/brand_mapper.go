package mapper

import (
	"smart-wardrobe-be/services/commerce/internal/modules/dashboard/application/dto"
	"smart-wardrobe-be/shared/domain/constants/brandmemberrole"
)

func RedactPointsLiabilityForRole(res *dto.BrandPointsLiabilityRes, role brandmemberrole.BrandMemberRole) *dto.BrandPointsLiabilityRes {
	if res == nil {
		return nil
	}

	// Redact financial values for non-Owner (e.g. Staff)
	if role != brandmemberrole.Owner {
		return &dto.BrandPointsLiabilityRes{
			BrandID:                 res.BrandID,
			PointsExpiring30d:       res.PointsExpiring30d,
			PointsLiabilityValueVND: 0, // Redacted for Staff
		}
	}

	return res
}
