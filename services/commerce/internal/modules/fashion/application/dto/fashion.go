package dto

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type FashionMetadataResult struct {
	CategorySlug string `json:"category_slug"`
	Color        string `json:"color"`
	ColorHex     string `json:"color_hex"`
	Style        string `json:"style"`
	Material     string `json:"material"`
	Pattern      string `json:"pattern"`
	Fit          string `json:"fit"`
	Seasonality  string `json:"seasonality"`
	Description  string `json:"description"`
	IsSingleItem bool   `json:"is_single_item"`
	// CategoryConfident is only meaningful when IsSingleItem is true. It is false
	// by default (safe): a missing signal routes the item to needs_review.
	CategoryConfident bool   `json:"category_confident"`
	ReviewReason      string `json:"review_reason"`
}

func (f *FashionMetadataResult) UnmarshalJSON(data []byte) error {
	type Alias FashionMetadataResult
	aux := struct {
		IsSingleItem      any `json:"is_single_item"`
		CategoryConfident any `json:"category_confident"`
		*Alias
	}{
		Alias: (*Alias)(f),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	switch v := aux.IsSingleItem.(type) {
	case bool:
		f.IsSingleItem = v
	case string:
		switch strings.ToLower(v) {
		case "true", "1", "yes":
			f.IsSingleItem = true
		case "false", "0", "no", "":
			f.IsSingleItem = false
		default:
			return fmt.Errorf("invalid boolean string for is_single_item: %s", v)
		}
	case nil:
		f.IsSingleItem = false
	default:
		return fmt.Errorf("unexpected type for is_single_item: %T", v)
	}

	switch v := aux.CategoryConfident.(type) {
	case bool:
		f.CategoryConfident = v
	case string:
		switch strings.ToLower(v) {
		case "true", "1", "yes":
			f.CategoryConfident = true
		case "false", "0", "no", "":
			f.CategoryConfident = false
		default:
			return fmt.Errorf("invalid boolean string for category_confident: %s", v)
		}
	case nil:
		f.CategoryConfident = false
	default:
		return fmt.Errorf("unexpected type for category_confident: %T", v)
	}
	return nil
}

type AICategoryRef struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type CategoryBriefRes struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
	Slug string    `json:"slug"`
}

type FashionItemBriefRes struct {
	ID       uuid.UUID         `json:"id"`
	Category *CategoryBriefRes `json:"category,omitempty"`
	ImageUrl string            `json:"imageUrl,omitempty"`
	Color    *string           `json:"color,omitempty"`
	ColorHex *string           `json:"colorHex,omitempty"`
	Style    *string           `json:"style,omitempty"`
	// Reasons are owned by the fashion item and only surfaced here (FR-031).
	ReviewReason          *string `json:"reviewReason,omitempty"`
	ProcessingErrorReason *string `json:"processingErrorReason,omitempty"`
}

type FashionProcessingInfoDTO struct {
	ID                      uuid.UUID  `json:"id"`
	ProcessingVersion       int        `json:"processingVersion"`
	ProcessingStartedAt     *time.Time `json:"processingStartedAt,omitempty"`
	LastProcessingAttemptAt *time.Time `json:"lastProcessingAttemptAt,omitempty"`
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
	// PreserveCategory keeps the item's existing category on re-analysis instead
	// of letting the AI overwrite a user-edited category (feature 023, FR-008).
	PreserveCategory bool `json:"preserveCategory,omitempty"`
	// FixedCategory runs a metadata-only analysis at the known category and skips
	// the is_single_item/category_confident gating so a needs_review item never
	// regenerates needs_review (feature 023, FR-038).
	FixedCategory bool `json:"fixedCategory,omitempty"`
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
