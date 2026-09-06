package dto

import (
	"time"

	"smart-wardrobe-be/internal/modules/brand/domain/constants/branditem/branditemstatus"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/branditem/branditemtype"
	fashion_dto "smart-wardrobe-be/internal/modules/fashion/application/dto"
	identity_dto "smart-wardrobe-be/internal/modules/identity/application/dto"
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

type BrandItemRes struct {
	ID            uuid.UUID                           `json:"id"`
	BrandID       uuid.UUID                           `json:"brandId"`
	BrandName     string                              `json:"brandName,omitempty"`
	FashionItemID uuid.UUID                           `json:"fashionItemId"`
	ProductCode   *string                             `json:"productCode"`
	Name          string                              `json:"name"`
	Description   *string                             `json:"description"`
	Price         *float64                            `json:"price"`
	ItemType      branditemtype.BrandItemType         `json:"itemType"`
	Status        branditemstatus.BrandItemStatus     `json:"status"`
	FashionItem   *fashion_dto.FashionItemContractDTO `json:"fashionItem,omitempty"`
	CreatedAt     time.Time                           `json:"createdAt"`
	UpdatedAt     time.Time                           `json:"updatedAt"`
	TaskID        *uuid.UUID                          `json:"taskId,omitempty"`
}

type SubmitSampleFeedbackReq struct {
	OutfitID     *uuid.UUID `json:"outfitId"`
	VoteType     *string    `json:"voteType" binding:"omitempty,oneof=like dislike would_buy not_interested" label:"loại vote"`
	Rating       *int       `json:"rating" binding:"omitempty,min=1,max=5" label:"điểm đánh giá"`
	FeedbackText *string    `json:"feedbackText" binding:"omitempty,max=1000" label:"phản hồi"`
}

type DigitalSampleResponseRes struct {
	ID           uuid.UUID                  `json:"id"`
	BrandItemID  uuid.UUID                  `json:"brandItemId"`
	UserID       uuid.UUID                  `json:"userId"`
	OutfitID     *uuid.UUID                 `json:"outfitId"`
	VoteType     *string                    `json:"voteType"`
	Rating       *int                       `json:"rating"`
	FeedbackText *string                    `json:"feedbackText"`
	User         *identity_dto.UserBriefRes `json:"user,omitempty"`
	CreatedAt    time.Time                  `json:"createdAt"`
}

type ListEligibleBrandItemsReq struct {
}

type GetBrandItemsQueryReq struct {
	shared_dto.PaginationQuery
}

type BrandItemListRes = shared_dto.PaginationResult[*BrandItemRes]

type BrandItemListBriefRes = shared_dto.PaginationResult[*BrandItemBriefRes]

type BrandItemBriefRes struct {
	ID            uuid.UUID                        `json:"id"`
	FashionItemID uuid.UUID                        `json:"-"`
	Brand         *BrandBriefRes                   `json:"brand,omitempty"`
	ProductCode   *string                          `json:"productCode,omitempty"`
	Name          string                           `json:"name"`
	Price         *float64                         `json:"price,omitempty"`
	ItemType      branditemtype.BrandItemType      `json:"itemType"`
	Status        branditemstatus.BrandItemStatus  `json:"status"`
	FashionItem   *fashion_dto.FashionItemBriefRes `json:"fashionItem,omitempty"`
}
