package usecase

import "context"

type IStatsSyncUseCase interface {
	ReconcileFollowStats(ctx context.Context) error
	ReconcilePostCounters(ctx context.Context) error
}
