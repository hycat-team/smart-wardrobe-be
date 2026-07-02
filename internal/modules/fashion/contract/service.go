package contract

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type FashionItemRes struct {
	ID              uuid.UUID  `json:"id"`
	CategoryID      *uuid.UUID `json:"categoryId,omitempty"`
	CategoryName    *string    `json:"categoryName,omitempty"`
	ImageUrl        string     `json:"imageUrl"`
	ImagePublicID   string     `json:"imagePublicId"`
	Color           *string    `json:"color,omitempty"`
	ColorHex        *string    `json:"colorHex,omitempty"`
	ColorHue        *float64   `json:"colorHue,omitempty"`
	ColorSaturation *float64   `json:"colorSaturation,omitempty"`
	ColorLightness  *float64   `json:"colorLightness,omitempty"`
	Style           *string    `json:"style,omitempty"`
	Material        *string    `json:"material,omitempty"`
	Pattern         *string    `json:"pattern,omitempty"`
	Fit             *string    `json:"fit,omitempty"`
	Seasonality     *string    `json:"seasonality,omitempty"`
	Description     *string    `json:"description,omitempty"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}

type FashionAnalyzeJobDTO struct {
	FashionItemID     uuid.UUID  `json:"fashionItemId"`
	ItemID            uuid.UUID  `json:"itemId"`   // WardrobeItem ID or BrandItem ID
	ItemType          string     `json:"itemType"` // "wardrobe" or "brand"
	UserID            uuid.UUID  `json:"userId"`
	ImageUrl          string     `json:"imageUrl"`
	ImagePublicID     string     `json:"imagePublicId"`
	CategoryID        *uuid.UUID `json:"categoryId,omitempty"`
	ProcessingVersion int        `json:"processingVersion"`
	RetryCount        int        `json:"retryCount,omitempty"`
	SkipVision        bool       `json:"skipVision,omitempty"`
}

type IFashionContract interface {
	CreateFashionItem(ctx context.Context, userID uuid.UUID, itemID uuid.UUID, itemType string, categoryID *uuid.UUID, imageUrl string, imagePublicID string) (uuid.UUID, error)
	GetFashionItem(ctx context.Context, id uuid.UUID) (*FashionItemRes, error)
	ListFashionItems(ctx context.Context, ids []uuid.UUID) ([]*FashionItemRes, error)
}
