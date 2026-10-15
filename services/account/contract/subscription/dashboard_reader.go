package contract

import (
	"context"
	"time"

	"smart-wardrobe-be/services/account/dto/subscription"
	uc_interfaces "smart-wardrobe-be/services/account/internal/modules/subscription/application/interface/usecase"
)

type ISubscriptionDashboardReader interface {
	// The period methods take the half-open VN range [from, to) (029 R1/
	// data-model §3); a zero bound leaves that side open. For the stats the
	// wire request also carries the legacy snapshot_date: when both bounds are
	// zero the server falls back to that day's range, keeping old clients
	// working.
	GetAdminSubscriptionStats(ctx context.Context, from, to time.Time) (*dto.AdminSubscriptionStatsDTO, error)
	GetSubscriptionDistribution(ctx context.Context, from, to time.Time) ([]dto.SubscriptionDistributionDTO, error)
	GetRevenueBreakdown(ctx context.Context, fromDate, toDate time.Time) ([]dto.RevenueBreakdownDTO, error)
	GetRenewalStats(ctx context.Context, from, to time.Time) (*dto.RenewalStatsDTO, error)
}

type SubscriptionDashboardReader struct {
	adminStatsUC uc_interfaces.IAdminStatsUseCase
}

func NewSubscriptionDashboardReader(adminStatsUC uc_interfaces.IAdminStatsUseCase) ISubscriptionDashboardReader {
	return &SubscriptionDashboardReader{adminStatsUC: adminStatsUC}
}

func (r *SubscriptionDashboardReader) GetAdminSubscriptionStats(ctx context.Context, from, to time.Time) (*dto.AdminSubscriptionStatsDTO, error) {
	return r.adminStatsUC.GetAdminSubscriptionStats(ctx, from, to)
}

func (r *SubscriptionDashboardReader) GetSubscriptionDistribution(ctx context.Context, from, to time.Time) ([]dto.SubscriptionDistributionDTO, error) {
	return r.adminStatsUC.GetSubscriptionDistribution(ctx, from, to)
}

func (r *SubscriptionDashboardReader) GetRevenueBreakdown(ctx context.Context, fromDate, toDate time.Time) ([]dto.RevenueBreakdownDTO, error) {
	return r.adminStatsUC.GetRevenueBreakdown(ctx, fromDate, toDate)
}

func (r *SubscriptionDashboardReader) GetRenewalStats(ctx context.Context, from, to time.Time) (*dto.RenewalStatsDTO, error) {
	return r.adminStatsUC.GetRenewalStats(ctx, from, to)
}
