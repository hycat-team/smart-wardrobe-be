package contract

import (
	"context"
	"smart-wardrobe-be/internal/shared/domain/entities"

	"github.com/google/uuid"
)

type FashionAnalyzeJobDTO struct {
	FashionItemID     uuid.UUID  `json:"fashionItemId"`
	ItemID            uuid.UUID  `json:"itemId"`   // WardrobeItem ID or BrandItem ID
	ItemType          string     `json:"itemType"` // "wardrobe" or "brand"
	UserID            uuid.UUID  `json:"userId"`
	ImageUrl          string     `json:"imageUrl"`
	ImagePublicID     string     `json:"imagePublicID"`
	CategoryID        *uuid.UUID `json:"categoryId"`
	ProcessingVersion int        `json:"processingVersion"`
	RetryCount        int        `json:"retryCount"`
}

type IFashionContract interface {
	CreateFashionItem(ctx context.Context, userID uuid.UUID, itemID uuid.UUID, itemType string, categoryID *uuid.UUID, imageUrl string, imagePublicID string) (uuid.UUID, error)
	GetFashionItem(ctx context.Context, id uuid.UUID) (*entities.FashionItem, error)
	ListFashionItems(ctx context.Context, ids []uuid.UUID) ([]*entities.FashionItem, error)
}
