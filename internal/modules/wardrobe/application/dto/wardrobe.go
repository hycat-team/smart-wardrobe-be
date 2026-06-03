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
	Status        wardrobestatus.WardrobeItemStatus `json:"status"`
	IsLocked      bool                              `json:"isLocked"`
	CreatedAt     time.Time                         `json:"createdAt"`
	// Description   string                            `json:"description"`
}

type CloneWardrobeItemReq struct {
	Quantity int `json:"quantity" binding:"required,min=1,max=5"`
}

type InitClosetFromCatalogReq struct {
	CatalogItemIDs []uuid.UUID `json:"catalogItemIds" binding:"required,min=1"`
}

type WardrobeBatchUploadItemReq struct {
	CategoryID    *uuid.UUID `json:"categoryId,omitempty"`
	ImageUrl      string     `json:"imageUrl" binding:"required"`
	ImagePublicID string     `json:"imagePublicId" binding:"required"`
}

type BatchUploadWardrobeItemsReq struct {
	Items []WardrobeBatchUploadItemReq `json:"items" binding:"required,min=1"`
}

type WardrobeBatchUploadJobDTO struct {
	ItemID        uuid.UUID  `json:"itemId"`
	UserID        uuid.UUID  `json:"userId"`
	CategoryID    *uuid.UUID `json:"categoryId,omitempty"`
	ImageUrl      string     `json:"imageUrl"`
	ImagePublicID string     `json:"imagePublicId"`
	RetryCount    int        `json:"retryCount,omitempty"`
}

type WardrobeEventPayload struct {
	ItemID uuid.UUID `json:"item_id"`
	UserID uuid.UUID `json:"user_id"`
	Action string    `json:"action"` // "created", "updated", "deleted"
}

type SearchWardrobeItemRes struct {
	ID            uuid.UUID    `json:"id"`
	Category      *CategoryRes `json:"category,omitempty"`
	ImageUrl      string       `json:"imageUrl"`
	ImagePublicID string       `json:"imagePublicId"`
	Color         string       `json:"color"`
	Style         string       `json:"style"`
	Material      string       `json:"material"`
	Pattern       string       `json:"pattern"`
	Fit           string       `json:"fit"`
	Seasonality   string       `json:"seasonality"`
	IsSystem      bool         `json:"isSystem"`
	// Description   string       `json:"description"`
}

type ManualClassifyReq struct {
	CategoryID  uuid.UUID `json:"categoryId" binding:"required"`
	Color       string    `json:"color" binding:"required"`
	Style       string    `json:"style" binding:"required"`
	Material    string    `json:"material" binding:"required"`
	Pattern     string    `json:"pattern" binding:"required"`
	Fit         string    `json:"fit" binding:"required"`
	Seasonality string    `json:"seasonality" binding:"required"`
}
