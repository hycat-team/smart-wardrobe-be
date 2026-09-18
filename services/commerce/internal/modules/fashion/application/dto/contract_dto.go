package dto

import (
	"time"

	"github.com/google/uuid"
)

type FashionItemContractDTO struct {
	ID              uuid.UUID         `json:"id"`
	CategoryID      *uuid.UUID        `json:"category_id,omitempty"`
	Category        *CategoryBriefRes `json:"category,omitempty"`
	ImageUrl        string            `json:"image_url"`
	ImagePublicID   string            `json:"-"`
	Color           *string           `json:"color,omitempty"`
	ColorHex        *string           `json:"color_hex,omitempty"`
	ColorHue        *float64          `json:"color_hue,omitempty"`
	ColorSaturation *float64          `json:"color_saturation,omitempty"`
	ColorLightness  *float64          `json:"color_lightness,omitempty"`
	Style           *string           `json:"style,omitempty"`
	Material        *string           `json:"material,omitempty"`
	Pattern         *string           `json:"pattern,omitempty"`
	Fit             *string           `json:"fit,omitempty"`
	Seasonality     *string           `json:"seasonality,omitempty"`
	Description     *string           `json:"description,omitempty"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

type CategoryContractDTO struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	SortOrder int       `json:"sort_order"`
}
