package dto

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	"smart-wardrobe-be/internal/shared/domain/constants/wardrobestatus"

	"github.com/google/uuid"
)

type CreateWardrobeItemReq struct {
	CategoryID    uuid.UUID `json:"categoryId" binding:"required" label:"danh mục"`
	ImageUrl      string    `json:"imageUrl" binding:"required" label:"đường dẫn ảnh"`
	ImagePublicID string    `json:"imagePublicId" binding:"required" label:"mã ảnh"`
}

type WardrobeItemRes struct {
	ID                    uuid.UUID                         `json:"id"`
	UserID                uuid.UUID                         `json:"userId"`
	Category              *CategoryRes                      `json:"category,omitempty"`
	ImageUrl              string                            `json:"imageUrl"`
	ImagePublicID         string                            `json:"imagePublicId"`
	Color                 string                            `json:"color"`
	ColorHex              string                            `json:"colorHex"`
	ColorHue              *float64                          `json:"colorHue,omitempty"`
	ColorSaturation       *float64                          `json:"colorSaturation,omitempty"`
	ColorLightness        *float64                          `json:"colorLightness,omitempty"`
	Style                 string                            `json:"style"`
	Material              string                            `json:"material"`
	Pattern               string                            `json:"pattern"`
	Fit                   string                            `json:"fit"`
	Seasonality           string                            `json:"seasonality"`
	Price                 *float64                          `json:"price,omitempty"`
	Status                wardrobestatus.WardrobeItemStatus `json:"status"`
	ReviewReason          *string                           `json:"reviewReason,omitempty"`
	ProcessingErrorReason *string                           `json:"processingErrorReason,omitempty"`
	IsLocked              bool                              `json:"isLocked"`
	CreatedAt             time.Time                         `json:"createdAt"`
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
	ItemID            uuid.UUID  `json:"itemId"`
	FashionItemID     uuid.UUID  `json:"fashionItemId,omitempty"`
	UserID            uuid.UUID  `json:"userId"`
	CategoryID        *uuid.UUID `json:"categoryId,omitempty"`
	ImageUrl          string     `json:"imageUrl"`
	ImagePublicID     string     `json:"imagePublicId"`
	RetryCount        int        `json:"retryCount,omitempty"`
	ProcessingVersion int        `json:"processingVersion"`
	ItemType          string     `json:"itemType,omitempty"`
}

type FashionMetadataResult struct {
	CategorySlug string `json:"category_slug"`
	Color        string `json:"color"`
	ColorHex     string `json:"color_hex"`
	Style        string `json:"style"`
	Material     string `json:"material"`
	Pattern      string `json:"pattern"`
	Fit          string `json:"fit"`
	Seasonality  string `json:"seasonality"`
	Description  string `json:"description"`
	IsSingleItem bool   `json:"is_single_item"`
	ReviewReason string `json:"review_reason"`
}

func (f *FashionMetadataResult) UnmarshalJSON(data []byte) error {
	type Alias FashionMetadataResult
	aux := struct {
		IsSingleItem any `json:"is_single_item"`
		*Alias
	}{
		Alias: (*Alias)(f),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	switch v := aux.IsSingleItem.(type) {
	case bool:
		f.IsSingleItem = v
	case string:
		switch strings.ToLower(v) {
		case "true", "1", "yes":
			f.IsSingleItem = true
		case "false", "0", "no", "":
			f.IsSingleItem = false
		default:
			return fmt.Errorf("invalid boolean string for is_single_item: %s", v)
		}
	case nil:
		f.IsSingleItem = false
	default:
		return fmt.Errorf("unexpected type for is_single_item: %T", v)
	}
	return nil
}

type AICategoryRef struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type WardrobeEventPayload struct {
	ItemID uuid.UUID `json:"item_id"`
	UserID uuid.UUID `json:"user_id"`
	Action string    `json:"action"` // "created", "updated", "deleted"
}

type SearchWardrobeItemRes struct {
	ID              uuid.UUID    `json:"id"`
	Category        *CategoryRes `json:"category,omitempty"`
	ImageUrl        string       `json:"imageUrl"`
	ImagePublicID   string       `json:"imagePublicId"`
	Color           string       `json:"color"`
	ColorHex        string       `json:"colorHex"`
	ColorHue        *float64     `json:"colorHue,omitempty"`
	ColorSaturation *float64     `json:"colorSaturation,omitempty"`
	ColorLightness  *float64     `json:"colorLightness,omitempty"`
	Style           string       `json:"style"`
	Material        string       `json:"material"`
	Pattern         string       `json:"pattern"`
	Fit             string       `json:"fit"`
	Seasonality     string       `json:"seasonality"`
	Price           *float64     `json:"price,omitempty"`
	IsSystem        bool         `json:"isSystem"`
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

type RetryWardrobeAnalysisReq struct{}

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
