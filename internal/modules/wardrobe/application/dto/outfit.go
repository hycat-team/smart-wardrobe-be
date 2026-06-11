package dto

import (
	"smart-wardrobe-be/internal/shared/domain/constants/outfitstatus"
	"time"

	"github.com/google/uuid"
)

type SaveOutfitReq struct {
	Name          string              `json:"name" binding:"required,max=255" label:"tên bắt buộc"`
	Description   *string             `json:"description" binding:"omitempty" label:"mô tả"`
	CoverImageUrl *string             `json:"coverImageUrl" binding:"omitempty,url" label:"đường dẫn ảnh bìa"`
	CoverPublicID *string             `json:"coverPublicId" binding:"omitempty,max=255" label:"mã ảnh bìa"`
	Items         []SaveOutfitItemReq `json:"items" binding:"required,dive" label:"danh sách món đồ"`
}

type SaveOutfitItemReq struct {
	WardrobeItemID uuid.UUID `json:"wardrobeItemId" binding:"required" label:"mã trang phục"`
	PositionX      float64   `json:"positionX" label:"vị trí X"`
	PositionY      float64   `json:"positionY" label:"vị trí Y"`
	Scale          float64   `json:"scale" binding:"required,min=0.1" label:"tỉ lệ hiển thị"`
	LayerOrder     int16     `json:"layerOrder" binding:"required" label:"thứ tự lớp"`
}

type OutfitRes struct {
	ID            uuid.UUID                 `json:"id"`
	UserID        uuid.UUID                 `json:"userId"`
	Name          string                    `json:"name"`
	Description   *string                   `json:"description"`
	CoverImageUrl *string                   `json:"coverImageUrl"`
	CoverPublicID *string                   `json:"coverPublicId"`
	Status        outfitstatus.OutfitStatus `json:"status"`
	CreatedAt     time.Time                 `json:"createdAt"`
	UpdatedAt     time.Time                 `json:"updatedAt"`
	Items         []*OutfitItemRes          `json:"items,omitempty"`
}

type OutfitItemRes struct {
	ID           uuid.UUID        `json:"id"`
	WardrobeItem *WardrobeItemRes `json:"wardrobeItem"`
	PositionX    float64          `json:"positionX"`
	PositionY    float64          `json:"positionY"`
	Scale        float64          `json:"scale"`
	LayerOrder   int16            `json:"layerOrder"`
}
