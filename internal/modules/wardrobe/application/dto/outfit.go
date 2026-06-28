package dto

import (
	"time"

	"smart-wardrobe-be/internal/shared/domain/constants/outfitstatus"

	"github.com/google/uuid"
)

type SaveOutfitReq struct {
	Name          string              `json:"name" binding:"required,max=255" label:"ten bat buoc"`
	Description   *string             `json:"description" binding:"omitempty" label:"mo ta"`
	CoverImageUrl *string             `json:"coverImageUrl" binding:"omitempty,url" label:"duong dan anh bia"`
	CoverPublicID *string             `json:"coverPublicId" binding:"omitempty,max=255" label:"ma anh bia"`
	Items         []SaveOutfitItemReq `json:"items" binding:"required,dive" label:"danh sach mon do"`
}

type SaveOutfitItemReq struct {
	FashionItemID uuid.UUID `json:"fashionItemId" binding:"required" label:"ma fashion item"`
	PositionX     float64   `json:"positionX" label:"vi tri X"`
	PositionY     float64   `json:"positionY" label:"vi tri Y"`
	Scale         float64   `json:"scale" binding:"required,min=0.1" label:"ti le hien thi"`
	LayerOrder    int16     `json:"layerOrder" binding:"required" label:"thu tu lop"`
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
	ID            uuid.UUID          `json:"id"`
	FashionItemID uuid.UUID          `json:"fashionItemId"`
	ItemContext   string             `json:"itemContext"`
	WardrobeItem  *WardrobeItemRes   `json:"wardrobeItem,omitempty"`
	BrandItem     *BrandItemBriefRes `json:"brandItem,omitempty"`
	PositionX     float64            `json:"positionX"`
	PositionY     float64            `json:"positionY"`
	Scale         float64            `json:"scale"`
	LayerOrder    int16              `json:"layerOrder"`
}
