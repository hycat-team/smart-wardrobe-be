package usecase

import (
	"context"

	"smart-wardrobe-be/internal/shared/observability/workerlog"
)

type IDashboardAggregatorUseCase interface {
	AggregateAdminSnapshots(ctx context.Context, run *workerlog.Run) error
	AggregateBrandSnapshots(ctx context.Context, run *workerlog.Run) error
}
