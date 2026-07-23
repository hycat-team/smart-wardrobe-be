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
