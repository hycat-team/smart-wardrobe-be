package usecase

import (
	"context"

	"smart-wardrobe-be/internal/modules/fashion/application/dto"

	"github.com/google/uuid"
)

type IFashionItemUseCase interface {
	GetFashionItem(ctx context.Context, id uuid.UUID) (*dto.FashionItemContractDTO, error)
	ListFashionItems(ctx context.Context, ids []uuid.UUID) ([]*dto.FashionItemContractDTO, error)
	CreateFashionItem(ctx context.Context, input dto.CreateFashionItemInput) error
	BulkCreateFashionItems(ctx context.Context, inputs []dto.CreateFashionItemInput) error
	UpdateFashionItemCategory(ctx context.Context, fashionItemID uuid.UUID, categoryID uuid.UUID) error
	GetStaleProcessingInfo(ctx context.Context, fashionItemIDs []uuid.UUID) ([]dto.FashionProcessingInfoDTO, error)
	SearchFashionItemIDs(ctx context.Context, query *string, categorySlug *string) ([]uuid.UUID, error)
	GetFashionItemIDsByCategory(ctx context.Context, categoryID uuid.UUID) ([]uuid.UUID, error)
	GetCategoryIDsByFashionItemIDs(ctx context.Context, fashionItemIDs []uuid.UUID) ([]dto.DashboardFashionItemCategoryDTO, error)
	ReassignCategoryForCatalogItems(ctx context.Context, fashionItemIDs []uuid.UUID, fromCategoryID uuid.UUID, toCategoryID uuid.UUID) error
	ResetProcessingState(ctx context.Context, fashionItemID uuid.UUID) error
}
