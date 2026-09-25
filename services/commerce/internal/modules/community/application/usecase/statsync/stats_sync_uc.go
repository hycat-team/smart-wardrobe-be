package statsync

import (
	"context"

	uc_interfaces "smart-wardrobe-be/services/commerce/internal/modules/community/application/interface/usecase"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/repositories"
)

type StatsSyncUseCase struct {
	statsRepo repositories.IUserFollowStatsRepository
}

func NewStatsSyncUseCase(statsRepo repositories.IUserFollowStatsRepository) uc_interfaces.IStatsSyncUseCase {
	return &StatsSyncUseCase{statsRepo: statsRepo}
}

func (uc *StatsSyncUseCase) ReconcileFollowStats(ctx context.Context) error {
	return uc.statsRepo.Reconcile(ctx)
}
