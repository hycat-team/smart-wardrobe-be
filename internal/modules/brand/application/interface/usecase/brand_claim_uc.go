package usecase

import (
	"context"

	"smart-wardrobe-be/internal/modules/brand/application/dto"

	"github.com/google/uuid"
)

type IBrandClaimUseCase interface {
	CreateBrandCustomerClaim(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, customerID uuid.UUID) (*dto.CreateClaimTokenRes, error)
	ListBrandCustomerClaims(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, customerID uuid.UUID) ([]*dto.ClaimTokenRes, error)
	RevokeBrandCustomerClaim(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, customerID uuid.UUID, claimID uuid.UUID, input dto.RevokeClaimTokenReq) (*dto.ClaimTokenRes, error)
	ClaimBrandCustomer(ctx context.Context, userID uuid.UUID, claimToken string, clientIP string) (*dto.BrandCustomerRes, error)
}
