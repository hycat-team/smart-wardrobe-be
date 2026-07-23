package contract

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/modules/subscription/application/dto"
	uc_interfaces "smart-wardrobe-be/internal/modules/subscription/application/interface/usecase"
)

type ISubscriptionDashboardReader interface {
	GetAdminSubscriptionStats(ctx context.Context, snapshotDate time.Time) (*dto.AdminSubscriptionStatsDTO, error)
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
