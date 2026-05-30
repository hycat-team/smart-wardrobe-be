package dto

import (
	"time"

	"smart-wardrobe-be/internal/shared/domain/constants/wardrobestatus"

	"github.com/google/uuid"
)

type CreateWardrobeItemReq struct {
	CategoryID    uuid.UUID `json:"categoryId" binding:"required"`
	ImageUrl      string    `json:"imageUrl" binding:"required"`
	ImagePublicID string    `json:"imagePublicId" binding:"required"`
}

type WardrobeItemRes struct {
	ID            uuid.UUID                         `json:"id"`
	UserID        uuid.UUID                         `json:"userId"`
	Category      *CategoryRes                      `json:"category,omitempty"`
	ImageUrl      string                            `json:"imageUrl"`
	ImagePublicID string                            `json:"imagePublicId"`
	Color         string                            `json:"color"`
	Style         string                            `json:"style"`
	Material      string                            `json:"material"`
	Pattern       string                            `json:"pattern"`
	Fit           string                            `json:"fit"`
	Seasonality   string                            `json:"seasonality"`
	Description   string                            `json:"description"`
	Status        wardrobestatus.WardrobeItemStatus `json:"status"`
	IsLocked      bool                              `json:"isLocked"`
	CreatedAt     time.Time                         `json:"createdAt"`
}

type CloneWardrobeItemReq struct {
	Quantity int `json:"quantity" binding:"required,min=1,max=5"`
}

type InitClosetFromCatalogReq struct {
	CatalogItemIDs []uuid.UUID `json:"catalogItemIds" binding:"required,min=1"`
}

type BatchCropWardrobeItemReq struct {
	CategoryID    uuid.UUID `json:"categoryId" binding:"required"`
	ImageUrl      string    `json:"imageUrl" binding:"required"`
	ImagePublicID string    `json:"imagePublicId" binding:"required"`
}

type BatchCropWardrobeItemsReq struct {
	Items []BatchCropWardrobeItemReq `json:"items" binding:"required,min=1"`
}

type BatchCropJobDTO struct {
	ItemID        uuid.UUID `json:"itemId"`
	UserID        uuid.UUID `json:"userId"`
	CategoryID    uuid.UUID `json:"categoryId"`
	ImageUrl      string    `json:"imageUrl"`
	ImagePublicID string    `json:"imagePublicId"`
}
