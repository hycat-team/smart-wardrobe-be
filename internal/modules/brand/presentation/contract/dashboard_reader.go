package contract

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/modules/brand/application/dto"
	uc_interfaces "smart-wardrobe-be/internal/modules/brand/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/brandmemberrole"

	"github.com/google/uuid"
)

type IBrandDashboardReader interface {
	GetAdminBrandStats(ctx context.Context, snapshotDate time.Time) (*dto.BrandAdminStatsDTO, error)
	GetBrandAnalyticsStats(ctx context.Context, brandID uuid.UUID, snapshotDate time.Time) (*dto.BrandAnalyticsStatsDTO, error)
	GetUserBrandRole(ctx context.Context, userID uuid.UUID, brandID uuid.UUID) (brandmemberrole.BrandMemberRole, error)
	GetSampleLabFeedbacksDrillDown(ctx context.Context, itemID uuid.UUID, page, limit int) ([]dto.SampleFeedbackDrillDownDTO, int64, error)
	GetActiveBrandIDs(ctx context.Context) ([]uuid.UUID, error)
}

type BrandDashboardReader struct {
	brandUC uc_interfaces.IBrandUseCase
}

func NewBrandDashboardReader(brandUC uc_interfaces.IBrandUseCase) IBrandDashboardReader {
	return &BrandDashboardReader{brandUC: brandUC}
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
