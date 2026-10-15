package usecase

import (
	"context"
	"time"

	"smart-wardrobe-be/services/commerce/internal/modules/dashboard/application/dto"
)

type IAdminDashboardUseCase interface {
	GetAdminKPICards(ctx context.Context) (*dto.AdminKPICardsRes, error)
	GetAIMarginAnalytics(ctx context.Context, fromDate, toDate time.Time) (*dto.AIMarginAnalyticsRes, error)
	// GetPayOSReconciliation takes the half-open VN day range [from, to)
	// (029 R6/data-model §3); both zero defaults to today (VN), a zero side
	// stays open.
	GetPayOSReconciliation(ctx context.Context, from, to time.Time) (*dto.PayOSReconciliationRes, error)
	GetSystemOverview(ctx context.Context) (*dto.AdminOverviewRes, error)

	// Expansion APIs
	GetSubscriptionDistribution(ctx context.Context, from, to time.Time) (*dto.SubscriptionDistributionRes, error)
	GetRevenueBreakdown(ctx context.Context, fromDate, toDate time.Time) (*dto.RevenueBreakdownRes, error)
	GetAIUsageByOperation(ctx context.Context, fromDate, toDate time.Time) (*dto.AIUsageByOperationRes, error)
	GetAIErrorBreakdown(ctx context.Context, from, to time.Time) (*dto.AIErrorBreakdownRes, error)
	GetRenewalStats(ctx context.Context, from, to time.Time) (*dto.RenewalStatsRes, error)
}
