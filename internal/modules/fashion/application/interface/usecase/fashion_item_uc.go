package usecase

import (
	"context"

	"smart-wardrobe-be/internal/modules/fashion/application/dto"

	"github.com/google/uuid"
)

type IFashionItemUseCase interface {
	GetFashionItem(ctx context.Context, id uuid.UUID) (*dto.FashionItemRes, error)
	ListFashionItems(ctx context.Context, ids []uuid.UUID) ([]*dto.FashionItemRes, error)
	CreateFashionItem(ctx context.Context, input dto.CreateFashionItemInput) error
	BulkCreateFashionItems(ctx context.Context, inputs []dto.CreateFashionItemInput) error
}
