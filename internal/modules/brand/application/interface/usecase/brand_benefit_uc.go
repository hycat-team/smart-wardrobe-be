package usecase

import (
	"context"

	"smart-wardrobe-be/internal/modules/brand/application/dto"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/benefit/benefitfeaturecode"

	"github.com/google/uuid"
)

type IBrandBenefitUseCase interface {
	CreateBrandBenefit(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, input dto.CreateBrandBenefitReq) (*dto.BrandBenefitRes, error)
	ListBrandBenefitsForStaff(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, query dto.GetBrandBenefitsQueryReq) (*dto.BrandBenefitListRes, error)
	ListActiveBenefitsForUser(ctx context.Context, brandID uuid.UUID, query dto.GetBrandBenefitsQueryReq) (*dto.BrandBenefitListRes, error)
	GetActiveBenefitForUser(ctx context.Context, userID uuid.UUID, benefitID uuid.UUID) (*dto.BrandBenefitRes, error)
	ListBenefitRedemptionsForUser(ctx context.Context, userID uuid.UUID, brandID *uuid.UUID, page, limit int) (*dto.BenefitRedemptionListRes, error)
	RedeemBenefit(ctx context.Context, userID uuid.UUID, benefitID uuid.UUID) (*dto.BenefitRedemptionRes, error)
	CheckBrandFeatureAccess(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, featureCode benefitfeaturecode.BenefitFeatureCode) (bool, error)
	UpdateBenefitStatus(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, benefitID uuid.UUID, status string) (*dto.BrandBenefitRes, error)
}
