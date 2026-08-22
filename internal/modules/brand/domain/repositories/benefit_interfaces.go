package repositories

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/modules/brand/domain/constants/benefit/benefittype"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/benefit/benefitunlocktype"
	"smart-wardrobe-be/internal/modules/brand/domain/entities"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"

	"github.com/google/uuid"
)

type BrandBenefitFilter struct {
	BrandID     uuid.UUID
	UnlockType  *benefitunlocktype.BenefitUnlockType
	BenefitType *benefittype.BenefitType
	Page        int
	Limit       int
}

type BrandBenefitListResult struct {
	Benefits   []*entities.BrandBenefit
	TotalCount int64
}

type IBrandBenefitRepository interface {
	shared_repos.IGenericRepository[entities.BrandBenefit, uuid.UUID]
	GetByBrandID(ctx context.Context, brandID uuid.UUID) ([]*entities.BrandBenefit, error)
	GetActiveByBrandID(ctx context.Context, brandID uuid.UUID) ([]*entities.BrandBenefit, error)
	GetActiveByBrandIDAndTierID(ctx context.Context, brandID, tierID uuid.UUID) ([]*entities.BrandBenefit, error)
	GetActiveByBrandIDPaginated(ctx context.Context, filter BrandBenefitFilter) (*BrandBenefitListResult, error)
}

type IBenefitRedemptionRepository interface {
	shared_repos.IGenericRepository[entities.BenefitRedemption, uuid.UUID]
	GetByBrandCustomerID(ctx context.Context, brandCustomerID uuid.UUID) ([]*entities.BenefitRedemption, error)
	GetByBrandCustomerIDs(ctx context.Context, brandCustomerIDs []uuid.UUID) ([]*entities.BenefitRedemption, error)
	GetByFilter(ctx context.Context, filter BenefitRedemptionFilter) (*BenefitRedemptionListResult, error)
	GetActiveRedemptionByFeature(ctx context.Context, brandCustomerID uuid.UUID, featureCode string, now time.Time) (*entities.BenefitRedemption, error)
	GetUserExpiringBenefitsCount(ctx context.Context, userID uuid.UUID) (int, error)
	GetRedemptionAnalyticsByBrand(ctx context.Context, brandID uuid.UUID) ([]BenefitRedemptionAnalytics, error)
	CountRedemptionsByBrand(ctx context.Context, brandID uuid.UUID, fromDate, toDate time.Time) (int64, error)
}

type BenefitRedemptionFilter struct {
	BrandCustomerIDs []uuid.UUID
	BrandID          *uuid.UUID
	Page             int
	Limit            int
}

type BenefitRedemptionListResult struct {
	Items      []*entities.BenefitRedemption
	TotalCount int64
}

type BenefitRedemptionAnalytics struct {
	BenefitID       uuid.UUID
	BenefitName     string
	BenefitType     string
	RedemptionCount int64
	Month           time.Time
}
