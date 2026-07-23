package usecase

import (
	"context"
)

type IDashboardAggregatorUseCase interface {
	AggregateAdminSnapshots(ctx context.Context) error
	AggregateBrandSnapshots(ctx context.Context) error
}
