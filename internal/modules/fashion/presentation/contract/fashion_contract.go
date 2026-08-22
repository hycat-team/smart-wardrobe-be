package contract

import (
	"context"

	"smart-wardrobe-be/internal/modules/fashion/application/dto"

	"github.com/google/uuid"
)

type IFashionContract interface {
	GetFashionItem(ctx context.Context, id uuid.UUID) (*dto.FashionItemRes, error)
	ListFashionItems(ctx context.Context, ids []uuid.UUID) ([]*dto.FashionItemRes, error)
	ListFashionItemsBrief(ctx context.Context, ids []uuid.UUID) ([]*dto.FashionItemBriefRes, error)
	CreateFashionItem(ctx context.Context, input dto.CreateFashionItemInput) error
	BulkCreateFashionItems(ctx context.Context, inputs []dto.CreateFashionItemInput) error
	UpdateFashionItemCategory(ctx context.Context, fashionItemID uuid.UUID, categoryID uuid.UUID) error
	GetStaleProcessingInfo(ctx context.Context, fashionItemIDs []uuid.UUID) ([]dto.FashionProcessingInfoDTO, error)
	SearchFashionItemIDs(ctx context.Context, query *string, categorySlug *string) ([]uuid.UUID, error)
	ResetProcessingState(ctx context.Context, fashionItemID uuid.UUID) error
}
