package usecase

import (
	"context"
	"smart-wardrobe-be/internal/shared/observability/workerlog"
)

type IPostHotnessUseCase interface {
	RefreshHotness(ctx context.Context, run *workerlog.Run) error
}
