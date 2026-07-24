package admin_stats

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/modules/subscription/application/dto"
	uc_interfaces "smart-wardrobe-be/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/subscription/domain/repositories"
)

type AdminStatsUseCase struct {
	depositRepo repositories.IDepositTransactionRepository
}

func NewAdminStatsUseCase(depositRepo repositories.IDepositTransactionRepository) uc_interfaces.IAdminStatsUseCase {
	return &AdminStatsUseCase{depositRepo: depositRepo}
}

func (uc *AdminStatsUseCase) GetAdminSubscriptionStats(ctx context.Context, snapshotDate time.Time) (*dto.AdminSubscriptionStatsDTO, error) {
	return uc.depositRepo.GetAdminSubscriptionStats(ctx, snapshotDate)
}
