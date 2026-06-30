package mapper

import (
	"time"

	"smart-wardrobe-be/internal/modules/brand/application/dto"
	"smart-wardrobe-be/internal/shared/domain/entities"
)

func MapClaimToken(claim *entities.BrandCustomerClaim) *dto.ClaimTokenRes {
	if claim == nil {
		return nil
	}
	status := "active"
	now := time.Now().UTC()
	if claim.ConsumedAt != nil {
		status = "consumed"
	} else if claim.RevokedAt != nil {
		status = "revoked"
	} else if now.After(claim.ExpiresAt) {
		status = "expired"
	}
	return &dto.ClaimTokenRes{
		ID:              claim.ID,
		BrandCustomerID: claim.BrandCustomerID,
		ExpiresAt:       claim.ExpiresAt,
		ConsumedAt:      claim.ConsumedAt,
		RevokedAt:       claim.RevokedAt,
		RevokedByUserID: claim.RevokedByUserID,
		RevokedReason:   claim.RevokedReason,
		Status:          status,
		CreatedAt:       claim.CreatedAt,
	}
}

func MapClaimTokens(claims []*entities.BrandCustomerClaim) []*dto.ClaimTokenRes {
	res := make([]*dto.ClaimTokenRes, 0, len(claims))
	for _, claim := range claims {
		res = append(res, MapClaimToken(claim))
	}
	return res
}
