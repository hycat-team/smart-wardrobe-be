package repositories

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/modules/brand/domain/entities"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"

	"github.com/google/uuid"
)

type IBrandBenefitRepository interface {
	shared_repos.IGenericRepository[entities.BrandBenefit, uuid.UUID]
	GetByBrandID(ctx context.Context, brandID uuid.UUID) ([]*entities.BrandBenefit, error)
	GetActiveByBrandID(ctx context.Context, brandID uuid.UUID) ([]*entities.BrandBenefit, error)
}

type IBenefitRedemptionRepository interface {
	shared_repos.IGenericRepository[entities.BenefitRedemption, uuid.UUID]
	GetByBrandCustomerID(ctx context.Context, brandCustomerID uuid.UUID) ([]*entities.BenefitRedemption, error)
	GetByBrandCustomerIDs(ctx context.Context, brandCustomerIDs []uuid.UUID) ([]*entities.BenefitRedemption, error)
	GetActiveRedemptionByFeature(ctx context.Context, brandCustomerID uuid.UUID, featureCode string, now time.Time) (*entities.BenefitRedemption, error)
	GetUserExpiringBenefitsCount(ctx context.Context, userID uuid.UUID) (int, error)
	GetRedemptionAnalyticsByBrand(ctx context.Context, brandID uuid.UUID) ([]BenefitRedemptionAnalytics, error)
	CountRedemptionsByBrand(ctx context.Context, brandID uuid.UUID, fromDate, toDate time.Time) (int64, error)
}

type BenefitRedemptionAnalytics struct {
	BenefitID       uuid.UUID
	BenefitName     string
	BenefitType     string
	RedemptionCount int64
	Month           time.Time
}
