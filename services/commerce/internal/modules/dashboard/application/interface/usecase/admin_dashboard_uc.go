package usecase

import (
	"context"
	"time"

	"smart-wardrobe-be/services/commerce/internal/modules/dashboard/application/dto"
)

type IAdminDashboardUseCase interface {
	GetAdminKPICards(ctx context.Context) (*dto.AdminKPICardsRes, error)
	GetAIMarginAnalytics(ctx context.Context, fromDate, toDate time.Time) (*dto.AIMarginAnalyticsRes, error)
	GetPayOSReconciliation(ctx context.Context) (*dto.PayOSReconciliationRes, error)
	GetSystemOverview(ctx context.Context) (*dto.AdminOverviewRes, error)

	// Expansion APIs
	GetSubscriptionDistribution(ctx context.Context) (*dto.SubscriptionDistributionRes, error)
	GetRevenueBreakdown(ctx context.Context, fromDate, toDate time.Time) (*dto.RevenueBreakdownRes, error)
	GetAIUsageByOperation(ctx context.Context, fromDate, toDate time.Time) (*dto.AIUsageByOperationRes, error)
	GetAIErrorBreakdown(ctx context.Context) (*dto.AIErrorBreakdownRes, error)
	GetRenewalStats(ctx context.Context) (*dto.RenewalStatsRes, error)
}
