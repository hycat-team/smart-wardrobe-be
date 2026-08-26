package event

import (
	"context"
	fashionDto "smart-wardrobe-be/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
)

type ISearchSyncEventConsumer interface {
	ConsumeEvents(ctx context.Context, handler func(ctx context.Context, event dto.WardrobeEventPayload) error) error
}

type IWardrobeFashionAnalyzedEventConsumer interface {
	ConsumeEvents(ctx context.Context, handler func(ctx context.Context, event fashionDto.FashionAnalyzedEventPayload) error) error
}
