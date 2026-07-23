package usecase

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/modules/dashboard/application/dto"
)

type IAdminDashboardUseCase interface {
	GetAdminKPICards(ctx context.Context) (*dto.AdminKPICardsRes, error)
	GetAIMarginAnalytics(ctx context.Context, fromDate, toDate time.Time) (*dto.AIMarginAnalyticsRes, error)
	GetPayOSReconciliation(ctx context.Context) (*dto.PayOSReconciliationRes, error)
	GetSystemOverview(ctx context.Context) (*dto.AdminOverviewRes, error)
}
