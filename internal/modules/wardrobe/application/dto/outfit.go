package dto

import (
	"time"

	"github.com/google/uuid"
)

type SaveOutfitReq struct {
	Name          string             `json:"name" binding:"required,max=255"`
	Description   *string            `json:"description" binding:"omitempty"`
	CoverImageUrl *string            `json:"cover_image_url" binding:"omitempty,url"`
	Items         []SaveOutfitItemReq `json:"items" binding:"required,dive"`
}

type SaveOutfitItemReq struct {
	WardrobeItemID uuid.UUID `json:"wardrobe_item_id" binding:"required"`
	PositionX      float64   `json:"position_x" binding:"required,min=0,max=1"`
	PositionY      float64   `json:"position_y" binding:"required,min=0,max=1"`
	Scale          float64   `json:"scale" binding:"required,min=0.1,max=10"`
	LayerOrder     int16     `json:"layer_order" binding:"required"`
}

type OutfitRes struct {
	ID            uuid.UUID        `json:"id"`
	UserID        uuid.UUID        `json:"user_id"`
	Name          string           `json:"name"`
	Description   *string          `json:"description"`
	CoverImageUrl *string          `json:"cover_image_url"`
	Status        int16            `json:"status"`
	CreatedAt     time.Time        `json:"created_at"`
	UpdatedAt     time.Time        `json:"updated_at"`
	Items         []*OutfitItemRes `json:"items,omitempty"`
}

type OutfitItemRes struct {
	ID           uuid.UUID        `json:"id"`
	WardrobeItem *WardrobeItemRes `json:"wardrobe_item"`
	PositionX    float64          `json:"position_x"`
	PositionY    float64          `json:"position_y"`
	Scale        float64          `json:"scale"`
	LayerOrder   int16            `json:"layer_order"`
}
