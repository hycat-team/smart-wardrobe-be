package usecase

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/modules/brand/application/dto"

	"github.com/google/uuid"
)

type IBrandDashboardAnalyticsUseCase interface {
	GetCustomerAcquisition(ctx context.Context, brandID uuid.UUID) ([]dto.CustomerAcquisitionDTO, error)
	GetLoyaltyTierDistribution(ctx context.Context, brandID uuid.UUID) ([]dto.LoyaltyTierDistributionDTO, error)
	GetPointExpiryForecast(ctx context.Context, brandID uuid.UUID) ([]dto.PointExpiryForecastDTO, error)
	GetBenefitRedemptionAnalytics(ctx context.Context, brandID uuid.UUID) ([]dto.BenefitRedemptionDTO, error)
	GetCatalogStats(ctx context.Context, brandID uuid.UUID) (*dto.CatalogStatsDTO, error)
	GetCustomerSpendSegmentation(ctx context.Context, brandID uuid.UUID) ([]dto.SpendSegmentDTO, error)
	GetBrandNewCustomerCount(ctx context.Context, brandID uuid.UUID, fromDate, toDate time.Time) (int64, error)
	GetBrandCustomerClaimRate(ctx context.Context, brandID uuid.UUID) (float64, error)
	GetBrandRedemptionCount(ctx context.Context, brandID uuid.UUID, fromDate, toDate time.Time) (int64, error)
	GetBrandItemCount(ctx context.Context, brandID uuid.UUID) (int64, error)
	GetBrandItemCountByType(ctx context.Context, brandID uuid.UUID) ([]dto.CatalogStatGroupDTO, error)
}
