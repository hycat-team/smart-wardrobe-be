package dto

import (
	"smart-wardrobe-be/internal/modules/fashion/domain/repositories"
	"time"

	"smart-wardrobe-be/internal/shared/application/constants/analyzetaskstatus"

	"github.com/google/uuid"
)

type CategoryBriefRes struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
	Slug string    `json:"slug"`
}

type FashionItemRes struct {
	ID              uuid.UUID         `json:"id"`
	Category        *CategoryBriefRes `json:"category,omitempty"`
	ImageUrl        string            `json:"imageUrl"`
	ImagePublicID   string            `json:"-"`
	Color           *string           `json:"color,omitempty"`
	ColorHex        *string           `json:"colorHex,omitempty"`
	ColorHue        *float64          `json:"colorHue,omitempty"`
	ColorSaturation *float64          `json:"colorSaturation,omitempty"`
	ColorLightness  *float64          `json:"colorLightness,omitempty"`
	Style           *string           `json:"style,omitempty"`
	Material        *string           `json:"material,omitempty"`
	Pattern         *string           `json:"pattern,omitempty"`
	Fit             *string           `json:"fit,omitempty"`
	Seasonality     *string           `json:"seasonality,omitempty"`
	Description     *string           `json:"description,omitempty"`
	CreatedAt       time.Time         `json:"createdAt"`
	UpdatedAt       time.Time         `json:"updatedAt"`
}

type FashionItemBriefRes struct {
	ID       uuid.UUID         `json:"id"`
	Category *CategoryBriefRes `json:"category,omitempty"`
	ImageUrl string            `json:"imageUrl,omitempty"`
	Color    *string           `json:"color,omitempty"`
	ColorHex *string           `json:"colorHex,omitempty"`
	Style    *string           `json:"style,omitempty"`
}

type FashionProcessingInfoDTO struct {
	ID                      uuid.UUID  `json:"id"`
	ProcessingVersion       int        `json:"processingVersion"`
	ProcessingStartedAt     *time.Time `json:"processingStartedAt,omitempty"`
	LastProcessingAttemptAt *time.Time `json:"lastProcessingAttemptAt,omitempty"`
}

type DashboardCategoryDTO struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type DashboardFashionItemCategoryDTO = repositories.DashboardFashionItemCategoryDTO

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
	TaskID            string     `json:"taskId,omitempty"`
	Total             int        `json:"total,omitempty"`
	Index             int        `json:"index,omitempty"`
}

type CreateFashionItemInput struct {
	ID                uuid.UUID
	ImageUrl          string
	ImagePublicID     string
	CategoryID        *uuid.UUID
	ProcessingVersion int
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

type FashionAnalyzedEventPayload struct {
	TaskID        string                              `json:"taskId"`
	ItemID        uuid.UUID                           `json:"itemId"`
	ItemType      string                              `json:"itemType"` // "wardrobe" | "brand"
	FashionItemID *uuid.UUID                          `json:"fashionItemId,omitempty"`
	Status        analyzetaskstatus.AnalyzeTaskStatus `json:"status"` // "completed" | "needs_review" | "failed"
	Total         int                                 `json:"total"`
	Index         int                                 `json:"index"`
	FashionBrief  *FashionItemBriefRes                `json:"fashionBrief,omitempty"`
	Error         string                              `json:"error,omitempty"`
}

type FashionItemCreatedPayload struct {
	ItemID        uuid.UUID `json:"itemId"`
	FashionItemID uuid.UUID `json:"fashionItemId"`
	ItemType      string    `json:"itemType"` // "wardrobe" | "brand"
	UserID        uuid.UUID `json:"userId,omitempty"`
}

type FashionProcessingExhaustedPayload struct {
	WardrobeItemID uuid.UUID `json:"wardrobeItemId"`
	FashionItemID  uuid.UUID `json:"fashionItemId"`
	Reason         string    `json:"reason"`
	UserID         uuid.UUID `json:"userId,omitempty"`
}
