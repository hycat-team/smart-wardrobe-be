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

type CategoryRes struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	SortOrder int       `json:"sortOrder"`
}

type FashionAnalyzeJobDTO struct {
	FashionItemID     *uuid.UUID `json:"fashionItemId,omitempty"`
	ItemID            uuid.UUID  `json:"itemId"`
	ItemType          string     `json:"itemType"`
	UserID            uuid.UUID  `json:"userId"`
	ImageUrl          string     `json:"imageUrl"`
	ImagePublicID     string     `json:"imagePublicId"`
	CategoryID        *uuid.UUID `json:"categoryId,omitempty"`
	ProcessingVersion int        `json:"processingVersion"`
	RetryCount        int        `json:"retryCount,omitempty"`
	SkipVision        bool       `json:"skipVision,omitempty"`
}

type CreateCategoryInput struct {
	Name      string
	Slug      string
	SortOrder int
}

type UpdateCategoryInput struct {
	Name      *string
	Slug      *string
	SortOrder *int
}

type IFashionContract interface {
	GetFashionItem(ctx context.Context, id uuid.UUID) (*FashionItemRes, error)
	ListFashionItems(ctx context.Context, ids []uuid.UUID) ([]*FashionItemRes, error)
	GetCategory(ctx context.Context, id uuid.UUID) (*CategoryRes, error)
	ListCategories(ctx context.Context, ids []uuid.UUID) ([]*CategoryRes, error)
	GetAllCategories(ctx context.Context) ([]*CategoryRes, error)
	GetCategoryBySlug(ctx context.Context, slug string) (*CategoryRes, error)
	GetCategoryByName(ctx context.Context, name string) (*CategoryRes, error)
	CreateCategory(ctx context.Context, input CreateCategoryInput) (*CategoryRes, error)
	UpdateCategory(ctx context.Context, id uuid.UUID, input UpdateCategoryInput) (*CategoryRes, error)
	DeleteCategory(ctx context.Context, id uuid.UUID) error
}
