package dto

import (
	"time"

	"smart-wardrobe-be/internal/shared/application/constants/analyzetaskstatus"

	"github.com/google/uuid"
)

type FashionItemRes struct {
	ID              uuid.UUID  `json:"id"`
	CategoryID      *uuid.UUID `json:"categoryId,omitempty"`
	CategoryName    *string    `json:"categoryName,omitempty"`
	CategorySlug    *string    `json:"categorySlug,omitempty"`
	ImageUrl        string     `json:"imageUrl"`
	ImagePublicID   string     `json:"-"`
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

type FashionItemBriefRes struct {
	ID           uuid.UUID `json:"id"`
	ImageUrl     string    `json:"imageUrl"`
	ColorHex     *string   `json:"colorHex,omitempty"`
	CategoryName *string   `json:"categoryName,omitempty"`
	CategorySlug *string   `json:"categorySlug,omitempty"`
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
	TaskID            string     `json:"taskId,omitempty"`
	Total             int        `json:"total,omitempty"`
	Index             int        `json:"index,omitempty"`
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

type SSEEventPayload struct {
	ItemID uuid.UUID                           `json:"itemId"`
	Status analyzetaskstatus.AnalyzeTaskStatus `json:"status"` // "completed" | "failed" | "needs_review"
	Total  int                                 `json:"total"`
	Index  int                                 `json:"index"`
	Data   any                                 `json:"data,omitempty"`
	Error  string                              `json:"error,omitempty"`
}
