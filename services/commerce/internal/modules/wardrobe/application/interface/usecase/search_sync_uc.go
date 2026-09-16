package usecase

import (
	"context"
	fashionDto "smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/shared/observability/workerlog"
	"smart-wardrobe-be/shared/pkg/logger"
)

type ISearchSyncUseCase interface {
	ProcessSyncEvent(ctx context.Context, eventPayload dto.WardrobeEventPayload, run *workerlog.Run) error
	HandleFashionProcessingExhausted(ctx context.Context, payload fashionDto.FashionProcessingExhaustedEventPayload, run *workerlog.Run) error
	TryInitialSync(ctx context.Context, run *workerlog.Run) (bool, error)
	IsOfflineError(err error) bool
	RunInitialSyncWithRetry(ctx context.Context, log logger.Interface)
	// SearchEnabled reports whether the search engine is switched on (020, FR-001).
	SearchEnabled() bool
}
