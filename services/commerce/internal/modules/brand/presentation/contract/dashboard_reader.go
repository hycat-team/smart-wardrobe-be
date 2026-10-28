package contract

import (
	"context"
	"time"

	"smart-wardrobe-be/services/commerce/internal/modules/brand/application/dto"
	uc_interfaces "smart-wardrobe-be/services/commerce/internal/modules/brand/application/interface/usecase"
	"smart-wardrobe-be/shared/domain/constants/brandmemberrole"

	"github.com/google/uuid"
)

type IBrandDashboardReader interface {
	ResolveBrandRef(ctx context.Context, ref string) (uuid.UUID, error)
	GetAdminBrandStats(ctx context.Context, snapshotDate time.Time) (*dto.BrandAdminStatsDTO, error)
	GetBrandAnalyticsStats(ctx context.Context, brandID uuid.UUID, snapshotDate time.Time) (*dto.BrandAnalyticsStatsDTO, error)
	GetUserBrandRole(ctx context.Context, userID uuid.UUID, brandID uuid.UUID) (brandmemberrole.BrandMemberRole, error)
	GetSampleLabFeedbacksDrillDown(ctx context.Context, itemID uuid.UUID, page, limit int) ([]dto.SampleFeedbackDrillDownDTO, int64, error)
	GetActiveBrandIDs(ctx context.Context) ([]uuid.UUID, error)
	GetItemFeedbackSummary(ctx context.Context, itemID uuid.UUID) (*dto.ItemFeedbackSummaryDTO, error)
	GetUserExpiringBenefitsCount(ctx context.Context, userID uuid.UUID) (int, error)
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

type BrandDashboardReader struct {
	brandUC     uc_interfaces.IBrandUseCase
	analyticsUC uc_interfaces.IBrandDashboardAnalyticsUseCase
}

func NewBrandDashboardReader(brandUC uc_interfaces.IBrandUseCase, analyticsUC uc_interfaces.IBrandDashboardAnalyticsUseCase) IBrandDashboardReader {
	return &BrandDashboardReader{brandUC: brandUC, analyticsUC: analyticsUC}
}

func (r *BrandDashboardReader) ResolveBrandRef(ctx context.Context, ref string) (uuid.UUID, error) {
	return r.brandUC.ResolveBrandRef(ctx, ref)
}

func (r *BrandDashboardReader) GetAdminBrandStats(ctx context.Context, snapshotDate time.Time) (*dto.BrandAdminStatsDTO, error) {
	return r.brandUC.GetAdminBrandStats(ctx, snapshotDate)
}

func (r *BrandDashboardReader) GetBrandAnalyticsStats(ctx context.Context, brandID uuid.UUID, snapshotDate time.Time) (*dto.BrandAnalyticsStatsDTO, error) {
	return r.brandUC.GetBrandAnalyticsStats(ctx, brandID, snapshotDate)
}

func (r *BrandDashboardReader) GetUserBrandRole(ctx context.Context, userID uuid.UUID, brandID uuid.UUID) (brandmemberrole.BrandMemberRole, error) {
	return r.brandUC.GetUserBrandRole(ctx, userID, brandID)
}

func (r *BrandDashboardReader) GetSampleLabFeedbacksDrillDown(ctx context.Context, itemID uuid.UUID, page, limit int) ([]dto.SampleFeedbackDrillDownDTO, int64, error) {
	return r.brandUC.GetSampleLabFeedbacksDrillDown(ctx, itemID, page, limit)
}

func (r *BrandDashboardReader) GetActiveBrandIDs(ctx context.Context) ([]uuid.UUID, error) {
	res, err := r.brandUC.GetActiveBrands(ctx, dto.GetActiveBrandsQueryReq{})
	if err != nil || res == nil {
		return nil, err
	}
	ids := make([]uuid.UUID, 0, len(res.Items))
	for _, b := range res.Items {
		ids = append(ids, b.ID)
	}
	return ids, nil
}

func (r *BrandDashboardReader) GetItemFeedbackSummary(ctx context.Context, itemID uuid.UUID) (*dto.ItemFeedbackSummaryDTO, error) {
	return r.brandUC.GetItemFeedbackSummary(ctx, itemID)
}

func (r *BrandDashboardReader) GetUserExpiringBenefitsCount(ctx context.Context, userID uuid.UUID) (int, error) {
	return r.brandUC.GetUserExpiringBenefitsCount(ctx, userID)
}

func (r *BrandDashboardReader) GetCustomerAcquisition(ctx context.Context, brandID uuid.UUID) ([]dto.CustomerAcquisitionDTO, error) {
	return r.analyticsUC.GetCustomerAcquisition(ctx, brandID)
}

func (r *BrandDashboardReader) GetLoyaltyTierDistribution(ctx context.Context, brandID uuid.UUID) ([]dto.LoyaltyTierDistributionDTO, error) {
	return r.analyticsUC.GetLoyaltyTierDistribution(ctx, brandID)
}

func (r *BrandDashboardReader) GetPointExpiryForecast(ctx context.Context, brandID uuid.UUID) ([]dto.PointExpiryForecastDTO, error) {
	return r.analyticsUC.GetPointExpiryForecast(ctx, brandID)
}

func (r *BrandDashboardReader) GetBenefitRedemptionAnalytics(ctx context.Context, brandID uuid.UUID) ([]dto.BenefitRedemptionDTO, error) {
	return r.analyticsUC.GetBenefitRedemptionAnalytics(ctx, brandID)
}

func (r *BrandDashboardReader) GetCatalogStats(ctx context.Context, brandID uuid.UUID) (*dto.CatalogStatsDTO, error) {
	return r.analyticsUC.GetCatalogStats(ctx, brandID)
}

func (r *BrandDashboardReader) GetCustomerSpendSegmentation(ctx context.Context, brandID uuid.UUID) ([]dto.SpendSegmentDTO, error) {
	return r.analyticsUC.GetCustomerSpendSegmentation(ctx, brandID)
}

func (r *BrandDashboardReader) GetBrandNewCustomerCount(ctx context.Context, brandID uuid.UUID, fromDate, toDate time.Time) (int64, error) {
	return r.analyticsUC.GetBrandNewCustomerCount(ctx, brandID, fromDate, toDate)
}

func (r *BrandDashboardReader) GetBrandCustomerClaimRate(ctx context.Context, brandID uuid.UUID) (float64, error) {
	return r.analyticsUC.GetBrandCustomerClaimRate(ctx, brandID)
}

func (r *BrandDashboardReader) GetBrandRedemptionCount(ctx context.Context, brandID uuid.UUID, fromDate, toDate time.Time) (int64, error) {
	return r.analyticsUC.GetBrandRedemptionCount(ctx, brandID, fromDate, toDate)
}

func (r *BrandDashboardReader) GetBrandItemCount(ctx context.Context, brandID uuid.UUID) (int64, error) {
	return r.analyticsUC.GetBrandItemCount(ctx, brandID)
}

func (r *BrandDashboardReader) GetBrandItemCountByType(ctx context.Context, brandID uuid.UUID) ([]dto.CatalogStatGroupDTO, error) {
	return r.analyticsUC.GetBrandItemCountByType(ctx, brandID)
}
