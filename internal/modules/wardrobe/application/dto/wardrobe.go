package dto

import (
	"time"

	"smart-wardrobe-be/internal/shared/domain/constants/wardrobestatus"

	"github.com/google/uuid"
)

type CreateWardrobeItemReq struct {
	CategoryID    uuid.UUID `json:"categoryId" binding:"required" label:"danh mục"`
	ImageUrl      string    `json:"imageUrl" binding:"required" label:"đường dẫn ảnh"`
	ImagePublicID string    `json:"imagePublicId" binding:"required" label:"mã ảnh"`
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
	Price         *float64                          `json:"price,omitempty"`
	Status        wardrobestatus.WardrobeItemStatus `json:"status"`
	IsLocked      bool                              `json:"isLocked"`
	CreatedAt     time.Time                         `json:"createdAt"`
}

type CloneWardrobeItemReq struct {
	Quantity int `json:"quantity" binding:"required,min=1,max=5" label:"số lượng bản sao"`
}

type InitClosetFromCatalogReq struct {
	CatalogItemIDs []uuid.UUID `json:"catalogItemIds" binding:"required,min=1" label:"danh sách trang phục mẫu"`
}

type WardrobeBatchUploadItemReq struct {
	CategoryID    *uuid.UUID `json:"categoryId,omitempty"`
	ImageUrl      string     `json:"imageUrl" binding:"required" label:"đường dẫn ảnh"`
	ImagePublicID string     `json:"imagePublicId" binding:"required" label:"mã ảnh"`
}

type BatchUploadWardrobeItemsReq struct {
	Items []WardrobeBatchUploadItemReq `json:"items" binding:"required,min=1" label:"danh sách trang phục"`
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
	Price         *float64     `json:"price,omitempty"`
	IsSystem      bool         `json:"isSystem"`
}

type GetWardrobeItemsQueryReq struct {
	CategorySlug string `form:"category_slug"`
}

type SearchWardrobeItemsQueryReq struct {
	Query        string `form:"q"`
	CategorySlug string `form:"category_slug"`
}

type ManualClassifyReq struct {
	CategoryID  uuid.UUID `json:"categoryId" binding:"required" label:"danh mục"`
	Color       string    `json:"color" binding:"required" label:"màu sắc"`
	Style       string    `json:"style" binding:"required" label:"phong cách"`
	Material    string    `json:"material" binding:"required" label:"chất liệu"`
	Pattern     string    `json:"pattern" binding:"required" label:"họa tiết"`
	Fit         string    `json:"fit" binding:"required" label:"dáng mặc"`
	Seasonality string    `json:"seasonality" binding:"required" label:"mùa phù hợp"`
	Price       *float64  `json:"price,omitempty"`
}
