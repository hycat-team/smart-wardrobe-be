package event

import (
	"context"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
)

type IBatchCropJobConsumer interface {
	ConsumeJobs(ctx context.Context, handler func(ctx context.Context, job dto.BatchCropJobDTO) error) error
}

type ISearchSyncEventConsumer interface {
	ConsumeEvents(ctx context.Context, handler func(ctx context.Context, event dto.WardrobeEventPayload) error) error
}
