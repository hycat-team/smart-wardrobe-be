package dto

import (
	"time"

	wardrobe_dto "smart-wardrobe-be/internal/modules/wardrobe/application/dto"

	"github.com/google/uuid"
)

type PostMediaReq struct {
	MediaType string  `json:"mediaType" binding:"required" label:"loại phương tiện"`
	MediaURL  string  `json:"mediaUrl" binding:"required,url" label:"đường dẫn phương tiện"`
	PublicID  *string `json:"publicId"`
	SortOrder int16   `json:"sortOrder"`
}

type CreatePostReq struct {
	PostType    string         `json:"postType" binding:"required" label:"loại bài đăng"`
	Title       *string        `json:"title"`
	Content     string         `json:"content" binding:"required" label:"nội dung"`
	ContactInfo *string        `json:"contactInfo"`
	TotalPrice  *float64       `json:"totalPrice"`
	ItemIDs     []uuid.UUID    `json:"itemIds"`
	Media       []PostMediaReq `json:"media"`
}

type UpdatePostItemsBuyerReq struct {
	BuyerUserID uuid.UUID `json:"buyerUserId" binding:"required" label:"người mua"`
}

type RemovePostItemsReq struct {
	PostItemIDs []uuid.UUID `json:"postItemIds" binding:"required" label:"danh sách món đồ bài đăng"`
}

type AddCommentReq struct {
	Content string `json:"content" binding:"required" label:"nội dung bình luận"`
}

type UpdateCommentReq struct {
	Content string `json:"content" binding:"required" label:"nội dung bình luận"`
}

type LikePostReq struct {
	IsLiked *bool `json:"isLiked" binding:"required" label:"trạng thái yêu thích"`
}

type PostRes struct {
	ID           uuid.UUID       `json:"id"`
	UserID       uuid.UUID       `json:"userId"`
	PostType     string          `json:"postType"`
	Title        *string         `json:"title"`
	Content      string          `json:"content"`
	ContactInfo  *string         `json:"contactInfo"`
	TotalPrice   float64         `json:"totalPrice"`
	LikeCount    int             `json:"likeCount"`
	CommentCount int             `json:"commentCount"`
	Items        []*PostItemRes  `json:"items,omitempty"`
	Media        []*PostMediaRes `json:"media,omitempty"`
	Comments     []*CommentRes   `json:"comments,omitempty"`
	CreatedAt    time.Time       `json:"createdAt"`
	UpdatedAt    time.Time       `json:"updatedAt"`
}

type PostItemRes struct {
	ID            uuid.UUID                     `json:"id"`
	Item          *wardrobe_dto.WardrobeItemRes `json:"item"`
	Price         float64                       `json:"price"`
	ItemCondition int16                         `json:"itemCondition"`
	Status        int16                         `json:"status"`
	BuyerUserID   *uuid.UUID                    `json:"buyerUserId"`
	TransferState int16                         `json:"transferState"`
	SoldAt        *time.Time                    `json:"soldAt"`
}

type PostMediaRes struct {
	ID        uuid.UUID `json:"id"`
	MediaType string    `json:"mediaType"`
	MediaURL  string    `json:"mediaUrl"`
	PublicID  *string   `json:"publicId"`
	SortOrder int16     `json:"sortOrder"`
}

type CommentRes struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"userId"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
}

type PendingTransferRes struct {
	PostItemID uuid.UUID                     `json:"postItemId"`
	Item       *wardrobe_dto.WardrobeItemRes `json:"item"`
	SellerName string                        `json:"sellerName"`
}
