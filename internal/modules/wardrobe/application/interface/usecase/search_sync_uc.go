package usecase

import (
	"context"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/shared/observability/workerlog"
)

type ISearchSyncUseCase interface {
	ProcessSyncEvent(ctx context.Context, eventPayload dto.WardrobeEventPayload, run *workerlog.Run) error
	TryInitialSync(ctx context.Context, run *workerlog.Run) (bool, error)
}
