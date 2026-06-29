package usecase

import (
	"context"

	"smart-wardrobe-be/internal/modules/brand/application/dto"

	"github.com/google/uuid"
)

type IBrandBenefitUseCase interface {
	CreateBrandBenefit(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, input dto.CreateBrandBenefitReq) (*dto.BrandBenefitRes, error)
	ListBrandBenefitsForStaff(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID) ([]*dto.BrandBenefitRes, error)
	ListActiveBenefitsForUser(ctx context.Context, userID uuid.UUID, brandID uuid.UUID) ([]*dto.BrandBenefitRes, error)
	GetActiveBenefitForUser(ctx context.Context, userID uuid.UUID, benefitID uuid.UUID) (*dto.BrandBenefitRes, error)
	ListBenefitRedemptionsForUser(ctx context.Context, userID uuid.UUID) ([]*dto.BenefitRedemptionRes, error)
	RedeemBenefit(ctx context.Context, userID uuid.UUID, benefitID uuid.UUID) (*dto.BenefitRedemptionRes, error)
	CheckBrandFeatureAccess(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, featureCode string) (bool, error)
	UpdateBenefitStatus(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, benefitID uuid.UUID, status string) (*dto.BrandBenefitRes, error)
}
