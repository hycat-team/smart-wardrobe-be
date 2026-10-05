package statsync

import (
	"context"

	uc_interfaces "smart-wardrobe-be/services/commerce/internal/modules/community/application/interface/usecase"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/repositories"
)

type StatsSyncUseCase struct {
	statsRepo repositories.IUserFollowStatsRepository
	postRepo  repositories.IPostRepository
}

func NewStatsSyncUseCase(statsRepo repositories.IUserFollowStatsRepository, postRepo repositories.IPostRepository) uc_interfaces.IStatsSyncUseCase {
	return &StatsSyncUseCase{statsRepo: statsRepo, postRepo: postRepo}
}

func (uc *StatsSyncUseCase) ReconcileFollowStats(ctx context.Context) error {
	return uc.statsRepo.Reconcile(ctx)
}

func (uc *StatsSyncUseCase) ReconcilePostCounters(ctx context.Context) error {
	return uc.postRepo.ReconcileCounters(ctx)
}
