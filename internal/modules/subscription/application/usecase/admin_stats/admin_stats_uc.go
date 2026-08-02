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
	userSubRepo repositories.IUserSubscriptionRepository
	renewalRepo repositories.ISubscriptionRenewalAttemptRepository
}

func NewAdminStatsUseCase(
	depositRepo repositories.IDepositTransactionRepository,
	userSubRepo repositories.IUserSubscriptionRepository,
	renewalRepo repositories.ISubscriptionRenewalAttemptRepository,
) uc_interfaces.IAdminStatsUseCase {
	return &AdminStatsUseCase{
		depositRepo: depositRepo,
		userSubRepo: userSubRepo,
		renewalRepo: renewalRepo,
	}
}

func (uc *AdminStatsUseCase) GetAdminSubscriptionStats(ctx context.Context, snapshotDate time.Time) (*dto.AdminSubscriptionStatsDTO, error) {
	return uc.depositRepo.GetAdminSubscriptionStats(ctx, snapshotDate)
}

func (uc *AdminStatsUseCase) GetSubscriptionDistribution(ctx context.Context) ([]dto.SubscriptionDistributionDTO, error) {
	return uc.userSubRepo.GetDistributionByPlanCode(ctx)
}

func (uc *AdminStatsUseCase) GetRevenueBreakdown(ctx context.Context, fromDate, toDate time.Time) ([]dto.RevenueBreakdownDTO, error) {
	return uc.depositRepo.GetRevenueBreakdown(ctx, fromDate, toDate)
}

func (uc *AdminStatsUseCase) GetRenewalStats(ctx context.Context) (*dto.RenewalStatsDTO, error) {
	return uc.renewalRepo.GetRenewalStats(ctx)
}
