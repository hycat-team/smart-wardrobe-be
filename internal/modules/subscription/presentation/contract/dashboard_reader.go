package contract

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/modules/subscription/application/dto"
	uc_interfaces "smart-wardrobe-be/internal/modules/subscription/application/interface/usecase"
)

type ISubscriptionDashboardReader interface {
	GetAdminSubscriptionStats(ctx context.Context, snapshotDate time.Time) (*dto.AdminSubscriptionStatsDTO, error)
	GetSubscriptionDistribution(ctx context.Context) ([]dto.SubscriptionDistributionDTO, error)
	GetRevenueBreakdown(ctx context.Context, fromDate, toDate time.Time) ([]dto.RevenueBreakdownDTO, error)
	GetRenewalStats(ctx context.Context) (*dto.RenewalStatsDTO, error)
}

type SubscriptionDashboardReader struct {
	adminStatsUC uc_interfaces.IAdminStatsUseCase
}

func NewSubscriptionDashboardReader(adminStatsUC uc_interfaces.IAdminStatsUseCase) ISubscriptionDashboardReader {
	return &SubscriptionDashboardReader{adminStatsUC: adminStatsUC}
}

func (r *SubscriptionDashboardReader) GetAdminSubscriptionStats(ctx context.Context, snapshotDate time.Time) (*dto.AdminSubscriptionStatsDTO, error) {
	return r.adminStatsUC.GetAdminSubscriptionStats(ctx, snapshotDate)
}

func (r *SubscriptionDashboardReader) GetSubscriptionDistribution(ctx context.Context) ([]dto.SubscriptionDistributionDTO, error) {
	return r.adminStatsUC.GetSubscriptionDistribution(ctx)
}

func (r *SubscriptionDashboardReader) GetRevenueBreakdown(ctx context.Context, fromDate, toDate time.Time) ([]dto.RevenueBreakdownDTO, error) {
	return r.adminStatsUC.GetRevenueBreakdown(ctx, fromDate, toDate)
}

func (r *SubscriptionDashboardReader) GetRenewalStats(ctx context.Context) (*dto.RenewalStatsDTO, error) {
	return r.adminStatsUC.GetRenewalStats(ctx)
}
