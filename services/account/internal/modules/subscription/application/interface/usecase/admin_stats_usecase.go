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
	// GetSubscriptionDistribution returns the plan buckets plus the cutover
	// and completeness metadata (031 FR-012); a bare slice can no longer say
	// "this period predates complete history".
	GetSubscriptionDistribution(ctx context.Context, from, to time.Time) (*dto.SubscriptionDistributionResultDTO, error)
	GetRevenueBreakdown(ctx context.Context, fromDate, toDate time.Time) ([]dto.RevenueBreakdownDTO, error)
	GetRenewalStats(ctx context.Context, from, to time.Time) (*dto.RenewalStatsDTO, error)
}
