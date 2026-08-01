package brand

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/modules/brand/application/dto"
	uc_interfaces "smart-wardrobe-be/internal/modules/brand/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/loyaltypointlotstatus"
	"smart-wardrobe-be/internal/modules/brand/domain/repositories"

	"github.com/google/uuid"
)

type BrandDashboardAnalyticsUseCase struct {
	customerRepo   repositories.IBrandCustomerRepository
	loyaltyRepo    repositories.ILoyaltyAccountRepository
	lotRepo        repositories.ILoyaltyPointLotRepository
	redemptionRepo repositories.IBenefitRedemptionRepository
	brandItemRepo  repositories.IBrandItemRepository
}

func NewBrandDashboardAnalyticsUseCase(
	customerRepo repositories.IBrandCustomerRepository,
	loyaltyRepo repositories.ILoyaltyAccountRepository,
	lotRepo repositories.ILoyaltyPointLotRepository,
	redemptionRepo repositories.IBenefitRedemptionRepository,
	brandItemRepo repositories.IBrandItemRepository,
) uc_interfaces.IBrandDashboardAnalyticsUseCase {
	return &BrandDashboardAnalyticsUseCase{
		customerRepo:   customerRepo,
		loyaltyRepo:    loyaltyRepo,
		lotRepo:        lotRepo,
		redemptionRepo: redemptionRepo,
		brandItemRepo:  brandItemRepo,
	}
}

func (uc *BrandDashboardAnalyticsUseCase) GetCustomerAcquisition(ctx context.Context, brandID uuid.UUID) ([]dto.CustomerAcquisitionDTO, error) {
	rows, err := uc.customerRepo.CountByJoinedSource(ctx, brandID)
	if err != nil {
		return nil, err
	}
	result := make([]dto.CustomerAcquisitionDTO, 0, len(rows))
	for _, row := range rows {
		result = append(result, dto.CustomerAcquisitionDTO{
			Source:        row.Source,
			CustomerCount: row.CustomerCount,
		})
	}
	return result, nil
}

func (uc *BrandDashboardAnalyticsUseCase) GetLoyaltyTierDistribution(ctx context.Context, brandID uuid.UUID) ([]dto.LoyaltyTierDistributionDTO, error) {
	rows, err := uc.loyaltyRepo.CountByTier(ctx, brandID)
	if err != nil {
		return nil, err
	}
	result := make([]dto.LoyaltyTierDistributionDTO, 0, len(rows))
	for _, row := range rows {
		result = append(result, dto.LoyaltyTierDistributionDTO{
			TierID:      row.TierID,
			TierName:    row.TierName,
			TierRank:    row.TierRank,
			MemberCount: row.MemberCount,
		})
	}
	return result, nil
}

func (uc *BrandDashboardAnalyticsUseCase) GetPointExpiryForecast(ctx context.Context, brandID uuid.UUID) ([]dto.PointExpiryForecastDTO, error) {
	rows, err := uc.lotRepo.SumExpiringPointsByMonth(ctx, brandID, loyaltypointlotstatus.Active)
	if err != nil {
		return nil, err
	}
	result := make([]dto.PointExpiryForecastDTO, 0, len(rows))
	for _, row := range rows {
		result = append(result, dto.PointExpiryForecastDTO{
			Month:          row.Month,
			PointsExpiring: row.PointsExpiring,
		})
	}
	return result, nil
}

func (uc *BrandDashboardAnalyticsUseCase) GetBenefitRedemptionAnalytics(ctx context.Context, brandID uuid.UUID) ([]dto.BenefitRedemptionDTO, error) {
	rows, err := uc.redemptionRepo.GetRedemptionAnalyticsByBrand(ctx, brandID)
	if err != nil {
		return nil, err
	}
	result := make([]dto.BenefitRedemptionDTO, 0, len(rows))
	for _, row := range rows {
		result = append(result, dto.BenefitRedemptionDTO{
			BenefitID:       row.BenefitID,
			BenefitName:     row.BenefitName,
			BenefitType:     row.BenefitType,
			RedemptionCount: row.RedemptionCount,
			Month:           row.Month,
		})
	}
	return result, nil
}

func (uc *BrandDashboardAnalyticsUseCase) GetCatalogStats(ctx context.Context, brandID uuid.UUID) (*dto.CatalogStatsDTO, error) {
	byStatus, err := uc.brandItemRepo.CountByStatus(ctx, brandID)
	if err != nil {
		return nil, err
	}
	byType, err := uc.brandItemRepo.CountByType(ctx, brandID)
	if err != nil {
		return nil, err
	}
	total, err := uc.brandItemRepo.CountByBrandID(ctx, brandID)
	if err != nil {
		return nil, err
	}

	statusGroups := make([]dto.CatalogStatGroupDTO, 0, len(byStatus))
	for _, row := range byStatus {
		statusGroups = append(statusGroups, dto.CatalogStatGroupDTO{Label: row.Status, Count: row.Count})
	}
	typeGroups := make([]dto.CatalogStatGroupDTO, 0, len(byType))
	for _, row := range byType {
		typeGroups = append(typeGroups, dto.CatalogStatGroupDTO{Label: row.ItemType, Count: row.Count})
	}

	return &dto.CatalogStatsDTO{
		TotalCount: total,
		ByStatus:   statusGroups,
		ByType:     typeGroups,
	}, nil
}

func (uc *BrandDashboardAnalyticsUseCase) GetCustomerSpendSegmentation(ctx context.Context, brandID uuid.UUID) ([]dto.SpendSegmentDTO, error) {
	rows, err := uc.loyaltyRepo.CountBySpendSegment(ctx, brandID)
	if err != nil {
		return nil, err
	}
	result := make([]dto.SpendSegmentDTO, 0, len(rows))
	for _, row := range rows {
		result = append(result, dto.SpendSegmentDTO{
			Segment:       row.Segment,
			CustomerCount: row.CustomerCount,
		})
	}
	return result, nil
}

func (uc *BrandDashboardAnalyticsUseCase) GetBrandNewCustomerCount(ctx context.Context, brandID uuid.UUID, fromDate, toDate time.Time) (int64, error) {
	return uc.customerRepo.CountNewByBrandID(ctx, brandID, fromDate, toDate)
}

func (uc *BrandDashboardAnalyticsUseCase) GetBrandCustomerClaimRate(ctx context.Context, brandID uuid.UUID) (float64, error) {
	total, err := uc.customerRepo.CountByBrandID(ctx, brandID)
	if err != nil {
		return 0, err
	}
	if total == 0 {
		return 0, nil
	}
	claimed, err := uc.customerRepo.CountClaimedByBrandID(ctx, brandID)
	if err != nil {
		return 0, err
	}
	return float64(claimed) / float64(total) * 100, nil
}

func (uc *BrandDashboardAnalyticsUseCase) GetBrandRedemptionCount(ctx context.Context, brandID uuid.UUID, fromDate, toDate time.Time) (int64, error) {
	return uc.redemptionRepo.CountRedemptionsByBrand(ctx, brandID, fromDate, toDate)
}

func (uc *BrandDashboardAnalyticsUseCase) GetBrandItemCount(ctx context.Context, brandID uuid.UUID) (int64, error) {
	return uc.brandItemRepo.CountByBrandID(ctx, brandID)
}

func (uc *BrandDashboardAnalyticsUseCase) GetBrandItemCountByType(ctx context.Context, brandID uuid.UUID) ([]dto.CatalogStatGroupDTO, error) {
	rows, err := uc.brandItemRepo.CountByType(ctx, brandID)
	if err != nil {
		return nil, err
	}
	result := make([]dto.CatalogStatGroupDTO, 0, len(rows))
	for _, row := range rows {
		result = append(result, dto.CatalogStatGroupDTO{Label: row.ItemType, Count: row.Count})
	}
	return result, nil
}
