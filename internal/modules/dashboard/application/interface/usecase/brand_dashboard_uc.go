package usecase

import (
	"context"

	"smart-wardrobe-be/internal/modules/brand/domain/constants/brandmemberrole"
	"smart-wardrobe-be/internal/modules/dashboard/application/dto"

	"github.com/google/uuid"
)

type IBrandDashboardUseCase interface {
	GetBrandKPICards(ctx context.Context, brandID uuid.UUID) (*dto.BrandKPICardsRes, error)
	GetPointsLiability(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, role brandmemberrole.BrandMemberRole) (*dto.BrandPointsLiabilityRes, error)
	GetDigitalSampleLabAnalytics(ctx context.Context, brandID uuid.UUID, itemID uuid.UUID) (*dto.DigitalSampleLabAnalyticsRes, error)

	// Expansion APIs
	GetCustomerAcquisition(ctx context.Context, brandID uuid.UUID) (*dto.CustomerAcquisitionRes, error)
	GetLoyaltyTierDistribution(ctx context.Context, brandID uuid.UUID) (*dto.TierDistributionRes, error)
	GetPointExpiryForecast(ctx context.Context, brandID uuid.UUID) (*dto.PointExpiryForecastRes, error)
	GetBenefitRedemptionAnalytics(ctx context.Context, brandID uuid.UUID) (*dto.BenefitRedemptionAnalyticsRes, error)
	GetCatalogStats(ctx context.Context, brandID uuid.UUID) (*dto.CatalogStatsRes, error)
	GetCustomerSpendSegmentation(ctx context.Context, brandID uuid.UUID) (*dto.SpendSegmentationRes, error)
}
