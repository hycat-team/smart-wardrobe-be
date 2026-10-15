package admin_stats

import (
	"context"
	"time"

	"smart-wardrobe-be/services/account/dto/subscription"
	uc_interfaces "smart-wardrobe-be/services/account/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/repositories"
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

func (uc *AdminStatsUseCase) GetAdminSubscriptionStats(ctx context.Context, from, to time.Time) (*dto.AdminSubscriptionStatsDTO, error) {
	return uc.depositRepo.GetAdminSubscriptionStats(ctx, from, to)
}

func (uc *AdminStatsUseCase) GetSubscriptionDistribution(ctx context.Context, from, to time.Time) ([]dto.SubscriptionDistributionDTO, error) {
	return uc.userSubRepo.GetDistributionByPlanCode(ctx, from, to)
}

func (uc *AdminStatsUseCase) GetRevenueBreakdown(ctx context.Context, fromDate, toDate time.Time) ([]dto.RevenueBreakdownDTO, error) {
	return uc.depositRepo.GetRevenueBreakdown(ctx, fromDate, toDate)
}

func (uc *AdminStatsUseCase) GetRenewalStats(ctx context.Context, from, to time.Time) (*dto.RenewalStatsDTO, error) {
	return uc.renewalRepo.GetRenewalStats(ctx, from, to)
}
