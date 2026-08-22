package dto

import (
	"time"

	"smart-wardrobe-be/internal/modules/wardrobe/domain/constants/wardrobestatus"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"

	"github.com/google/uuid"
)

type CreateWardrobeItemReq struct {
	CategoryID    uuid.UUID `json:"categoryId" binding:"required" label:"danh mục"`
	ImageUrl      string    `json:"imageUrl" binding:"required" label:"đường dẫn ảnh"`
	ImagePublicID string    `json:"imagePublicId" binding:"required" label:"mã ảnh"`
}

type CategoryBriefRes struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
	Slug string    `json:"slug"`
}

type FashionItemRes struct {
	ID              uuid.UUID         `json:"id"`
	Category        *CategoryBriefRes `json:"category,omitempty"`
	ImageUrl        string            `json:"imageUrl"`
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
	CreatedAt       time.Time         `json:"createdAt"`
	UpdatedAt       time.Time         `json:"updatedAt"`
}

type BrandItemBriefRes struct {
	ID        uuid.UUID `json:"id"`
	BrandID   uuid.UUID `json:"brandId"`
	BrandName string    `json:"brandName"`
	ItemType  string    `json:"itemType"`
	Name      string    `json:"name"`
	Price     *float64  `json:"price,omitempty"`
}

type WardrobeItemRes struct {
	ID          uuid.UUID                         `json:"id"`
	UserID      uuid.UUID                         `json:"userId"`
	Price       *float64                          `json:"price,omitempty"`
	Status      wardrobestatus.WardrobeItemStatus `json:"status"`
	IsLocked    bool                              `json:"isLocked"`
	ItemContext string                            `json:"itemContext,omitempty"`
	BrandItem   *BrandItemBriefRes                `json:"brandItem,omitempty"`
	FashionItem *FashionItemRes                   `json:"fashionItem,omitempty"`
	CreatedAt   time.Time                         `json:"createdAt"`
	LastUsedAt  *time.Time                        `json:"lastUsedAt,omitempty"`
	IsDeleted   bool                              `json:"isDeleted"`
	TaskID      *uuid.UUID                        `json:"taskId,omitempty"`
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

type WardrobeEventPayload struct {
	ItemID        uuid.UUID `json:"item_id"`
	UserID        uuid.UUID `json:"user_id"`
	FashionItemID uuid.UUID `json:"fashion_item_id,omitempty"`
	ItemType      int       `json:"item_type,omitempty"`
	Action        string    `json:"action"` // "created", "updated", "deleted"
}

type SearchWardrobeItemRes struct {
	ID          uuid.UUID       `json:"id"`
	Price       *float64        `json:"price,omitempty"`
	IsSystem    bool            `json:"isSystem"`
	FashionItem *FashionItemRes `json:"fashionItem,omitempty"`
}

type GetWardrobeItemsQueryReq struct {
	shared_dto.PaginationQuery
	CategorySlug string `form:"categorySlug"`
	Status       string `form:"status"`
}

type SearchWardrobeItemsQueryReq struct {
	shared_dto.PaginationQuery
	Query        string `form:"q"`
	CategorySlug string `form:"categorySlug"`
}

type UpdateWardrobeItemReq struct {
	Price      *float64   `json:"price,omitempty"`
	CategoryID *uuid.UUID `json:"categoryId,omitempty"`
}

type GetSystemCatalogItemsQueryReq struct {
	shared_dto.PaginationQuery
	Query        *string `form:"q"`
	CategorySlug *string `form:"categorySlug"`
}

type GetOutfitsQueryReq struct {
	shared_dto.PaginationQuery
}

type UpdateSystemCatalogItemReq struct {
	CategoryID  *uuid.UUID `json:"categoryId" label:"danh mục"`
	Color       *string    `json:"color" label:"màu sắc"`
	Style       *string    `json:"style" label:"phong cách"`
	Material    *string    `json:"material" label:"chất liệu"`
	Pattern     *string    `json:"pattern" label:"họa tiết"`
	Fit         *string    `json:"fit" label:"dáng mặc"`
	Seasonality *string    `json:"seasonality" label:"mùa phù hợp"`
	Price       *float64   `json:"price,omitempty"`
}

type BulkDeleteItemsReq struct {
	IDs []uuid.UUID `json:"ids" binding:"required,min=1"`
}

type WardrobeStatsRes struct {
	ActiveItemsCount int `json:"activeItemsCount"`
	OutfitsCount     int `json:"outfitsCount"`
}
