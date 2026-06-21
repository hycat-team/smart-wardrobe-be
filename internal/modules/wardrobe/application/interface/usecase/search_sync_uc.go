package usecase

import (
	"context"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
)

type ISearchSyncUseCase interface {
	ProcessSyncEvent(ctx context.Context, eventPayload dto.WardrobeEventPayload) error
	TryInitialSync(ctx context.Context) (bool, error)
}
