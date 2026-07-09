package dto

import (
	"time"

	"smart-wardrobe-be/internal/modules/brand/domain/constants/branditem/branditemstatus"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/branditem/branditemtype"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"

	"github.com/google/uuid"
)

type CreateBrandItemReq struct {
	CategoryID    *uuid.UUID `json:"categoryId" binding:"omitempty" label:"mã danh mục"`
	ImageUrl      string     `json:"imageUrl" binding:"required,url" label:"đường dẫn hình ảnh"`
	ImagePublicID string     `json:"imagePublicId" binding:"required" label:"mã hình ảnh"`
	ProductCode   *string    `json:"productCode" binding:"omitempty,max=100" label:"mã sản phẩm"`
	Name          string     `json:"name" binding:"required,max=255" label:"tên sản phẩm"`
	Description   *string    `json:"description" binding:"omitempty" label:"mô tả"`
	Price         *float64   `json:"price" binding:"omitempty,gt=0" label:"giá sản phẩm"`
	ItemType      string     `json:"itemType" binding:"required" label:"loại sản phẩm"`
	Status        string     `json:"status" binding:"omitempty" label:"trạng thái"`
}

type UpdateBrandItemReq struct {
	ProductCode *string  `json:"productCode" binding:"omitempty" label:"mã sản phẩm"`
	Name        string   `json:"name" binding:"required,max=255" label:"tên sản phẩm"`
	Description *string  `json:"description" binding:"omitempty" label:"mô tả"`
	Price       *float64 `json:"price" binding:"omitempty,gt=0" label:"giá sản phẩm"`
	Status      string   `json:"status" binding:"required" label:"trạng thái"`
}

type UpdateBrandItemStatusReq struct {
	Status branditemstatus.BrandItemStatus `json:"status" binding:"required" label:"trạng thái sản phẩm"`
}

type FashionItemRes struct {
	ID              uuid.UUID  `json:"id"`
	CategoryID      *uuid.UUID `json:"categoryId,omitempty"`
	CategoryName    *string    `json:"categoryName,omitempty"`
	CategorySlug    *string    `json:"categorySlug,omitempty"`
	ImageUrl        string     `json:"imageUrl"`
	Color           *string    `json:"color,omitempty"`
	ColorHex        *string    `json:"colorHex,omitempty"`
	ColorHue        *float64   `json:"colorHue,omitempty"`
	ColorSaturation *float64   `json:"colorSaturation,omitempty"`
	ColorLightness  *float64   `json:"colorLightness,omitempty"`
	Style           *string    `json:"style,omitempty"`
	Material        *string    `json:"material,omitempty"`
	Pattern         *string    `json:"pattern,omitempty"`
	Fit             *string    `json:"fit,omitempty"`
	Seasonality     *string    `json:"seasonality,omitempty"`
	Description     *string    `json:"description,omitempty"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}

type BrandItemRes struct {
	ID            uuid.UUID                       `json:"id"`
	BrandID       uuid.UUID                       `json:"brandId"`
	BrandName     string                          `json:"brandName,omitempty"`
	FashionItemID uuid.UUID                       `json:"fashionItemId"`
	ProductCode   *string                         `json:"productCode"`
	Name          string                          `json:"name"`
	Description   *string                         `json:"description"`
	Price         *float64                        `json:"price"`
	ItemType      branditemtype.BrandItemType     `json:"itemType"`
	Status        branditemstatus.BrandItemStatus `json:"status"`
	FashionItem   *FashionItemRes                 `json:"fashionItem,omitempty"`
	CreatedAt     time.Time                       `json:"createdAt"`
	UpdatedAt     time.Time                       `json:"updatedAt"`
	TaskID        *uuid.UUID                      `json:"taskId,omitempty"`
}

type SubmitSampleFeedbackReq struct {
	OutfitID     *uuid.UUID `json:"outfitId"`
	VoteType     *string    `json:"voteType" binding:"omitempty,oneof=like dislike would_buy not_interested" label:"loại vote"`
	Rating       *int       `json:"rating" binding:"omitempty,min=1,max=5" label:"điểm đánh giá"`
	FeedbackText *string    `json:"feedbackText" binding:"omitempty,max=1000" label:"phản hồi"`
}

type DigitalSampleResponseRes struct {
	ID           uuid.UUID  `json:"id"`
	BrandItemID  uuid.UUID  `json:"brandItemId"`
	UserID       uuid.UUID  `json:"userId"`
	OutfitID     *uuid.UUID `json:"outfitId"`
	VoteType     *string    `json:"voteType"`
	Rating       *int       `json:"rating"`
	FeedbackText *string    `json:"feedbackText"`
	CreatedAt    time.Time  `json:"createdAt"`
}

type ListEligibleBrandItemsReq struct {
}

type GetBrandItemsQueryReq struct {
	shared_dto.PaginationQuery
}

type BrandItemListRes = shared_dto.PaginationResult[*BrandItemRes]
