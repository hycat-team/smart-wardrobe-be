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

type IWardrobeCleanupEventConsumer interface {
	ConsumeEvents(ctx context.Context, handler func(ctx context.Context, event dto.WardrobeEventPayload) error) error
}

type IWardrobeProcessingExhaustedEventConsumer interface {
	ConsumeEvents(ctx context.Context, handler func(ctx context.Context, event fashionDto.FashionProcessingExhaustedPayload) error) error
}

type IWardrobeFashionBindEventConsumer interface {
	ConsumeEvents(ctx context.Context, handler func(ctx context.Context, event fashionDto.FashionItemCreatedPayload) error) error
}

type IElasticsearchProcessingExhaustedEventConsumer interface {
	ConsumeEvents(ctx context.Context, handler func(ctx context.Context, event fashionDto.FashionProcessingExhaustedPayload) error) error
}
