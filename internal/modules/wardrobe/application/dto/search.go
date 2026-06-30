package dto

import (
	"time"

	"github.com/google/uuid"
	"smart-wardrobe-be/internal/shared/domain/constants/wardrobe/itemtype"
	"smart-wardrobe-be/internal/shared/domain/constants/wardrobe/wardrobestatus"
)

type SearchDocumentCategoryDTO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type SearchDocumentDTO struct {
	ID              uuid.UUID                         `json:"id"`
	UserID          uuid.UUID                         `json:"user_id"`
	FashionItemID   uuid.UUID                         `json:"fashion_item_id"`
	ItemType        itemtype.ItemType                 `json:"item_type"`
	ImageUrl        string                            `json:"image_url"`
	ImagePublicID   string                            `json:"image_public_id"`
	Color           string                            `json:"color"`
	Style           string                            `json:"style"`
	Material        string                            `json:"material"`
	Pattern         string                            `json:"pattern"`
	Fit             string                            `json:"fit"`
	Seasonality     string                            `json:"seasonality"`
	Description     string                            `json:"description"`
	Status          wardrobestatus.WardrobeItemStatus `json:"status"`
	CreatedAt       time.Time                         `json:"created_at"`
	Category        SearchDocumentCategoryDTO         `json:"category"`
	Price           *float64                          `json:"price,omitempty"`
	ColorHex        *string                           `json:"color_hex,omitempty"`
	ColorHue        *float64                          `json:"color_hue,omitempty"`
	ColorSaturation *float64                          `json:"color_saturation,omitempty"`
	ColorLightness  *float64                          `json:"color_lightness,omitempty"`
}
