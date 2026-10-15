package usecase

import (
	"context"
	"time"

	"smart-wardrobe-be/services/account/dto/subscription"
)

type IAdminStatsUseCase interface {
	// The three period methods take the half-open VN range [from, to)
	// (data-model §3, R1); a zero bound leaves that side open and the account
	// repo keeps its current-state semantics when both are zero.
	GetAdminSubscriptionStats(ctx context.Context, from, to time.Time) (*dto.AdminSubscriptionStatsDTO, error)
	GetSubscriptionDistribution(ctx context.Context, from, to time.Time) ([]dto.SubscriptionDistributionDTO, error)
	GetRevenueBreakdown(ctx context.Context, fromDate, toDate time.Time) ([]dto.RevenueBreakdownDTO, error)
	GetRenewalStats(ctx context.Context, from, to time.Time) (*dto.RenewalStatsDTO, error)
}
