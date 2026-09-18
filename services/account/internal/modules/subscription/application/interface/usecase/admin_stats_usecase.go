package usecase

import (
	"context"
	"time"

	"smart-wardrobe-be/services/account/dto/subscription"
)

type IAdminStatsUseCase interface {
	GetAdminSubscriptionStats(ctx context.Context, snapshotDate time.Time) (*dto.AdminSubscriptionStatsDTO, error)
	GetSubscriptionDistribution(ctx context.Context) ([]dto.SubscriptionDistributionDTO, error)
	GetRevenueBreakdown(ctx context.Context, fromDate, toDate time.Time) ([]dto.RevenueBreakdownDTO, error)
	GetRenewalStats(ctx context.Context) (*dto.RenewalStatsDTO, error)
}
