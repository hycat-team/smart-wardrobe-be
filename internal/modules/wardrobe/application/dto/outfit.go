package dto

import (
	"time"

	"github.com/google/uuid"
)

type SaveOutfitReq struct {
	Name          string              `json:"name" binding:"required,max=255" label:"tên bộ phối"`
	Description   *string             `json:"description" binding:"omitempty" label:"mô tả"`
	CoverImageUrl *string             `json:"cover_image_url" binding:"omitempty,url" label:"đường dẫn ảnh bìa"`
	CoverPublicID *string             `json:"cover_public_id" binding:"omitempty,max=255" label:"mã ảnh bìa"`
	Items         []SaveOutfitItemReq `json:"items" binding:"required,dive" label:"danh sách món đồ"`
}

type SaveOutfitItemReq struct {
	WardrobeItemID uuid.UUID `json:"wardrobe_item_id" binding:"required" label:"mã trang phục"`
	PositionX      float64   `json:"position_x" binding:"required,min=0,max=1" label:"vị trí X"`
	PositionY      float64   `json:"position_y" binding:"required,min=0,max=1" label:"vị trí Y"`
	Scale          float64   `json:"scale" binding:"required,min=0.1,max=10" label:"tỷ lệ hiển thị"`
	LayerOrder     int16     `json:"layer_order" binding:"required" label:"thứ tự lớp"`
}

type OutfitRes struct {
	ID            uuid.UUID        `json:"id"`
	UserID        uuid.UUID        `json:"user_id"`
	Name          string           `json:"name"`
	Description   *string          `json:"description"`
	CoverImageUrl *string          `json:"cover_image_url"`
	CoverPublicID *string          `json:"cover_public_id"`
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
