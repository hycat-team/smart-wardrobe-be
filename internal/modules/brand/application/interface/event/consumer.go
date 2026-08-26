package event

import (
	"context"
	fashion_dto "smart-wardrobe-be/internal/modules/fashion/application/dto"
)

type IBrandFashionAnalyzedEventConsumer interface {
	ConsumeEvents(ctx context.Context, handler func(ctx context.Context, event fashion_dto.FashionAnalyzedEventPayload) error) error
}
