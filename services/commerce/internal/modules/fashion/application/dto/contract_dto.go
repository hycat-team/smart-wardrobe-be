package dto

import (
	"time"

	"github.com/google/uuid"
)

type FashionItemContractDTO struct {
	ID              uuid.UUID         `json:"id"`
	CategoryID      *uuid.UUID        `json:"categoryId,omitempty"`
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
	// Reasons surfaced to brand portal so staff can tell a system error from a
	// needs-review case (feature 023, US4).
	ProcessingErrorReason *string   `json:"processingErrorReason,omitempty"`
	ReviewReason          *string   `json:"reviewReason,omitempty"`
	CreatedAt             time.Time `json:"createdAt"`
	UpdatedAt             time.Time `json:"updatedAt"`
}

type CategoryContractDTO struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	SortOrder int       `json:"sortOrder"`
}
