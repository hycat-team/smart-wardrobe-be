package contract

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/modules/brand/application/dto"
	uc_interfaces "smart-wardrobe-be/internal/modules/brand/application/interface/usecase"

	"github.com/google/uuid"
)

type IBrandDashboardReader interface {
	GetAdminBrandStats(ctx context.Context, snapshotDate time.Time) (*dto.BrandAdminStatsDTO, error)
	GetBrandAnalyticsStats(ctx context.Context, brandID uuid.UUID, snapshotDate time.Time) (*dto.BrandAnalyticsStatsDTO, error)
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
