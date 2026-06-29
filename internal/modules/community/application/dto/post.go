package dto

import (
	"time"

	identity_dto "smart-wardrobe-be/internal/modules/identity/application/dto"
	wardrobe_dto "smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	"smart-wardrobe-be/internal/shared/domain/constants/community/postitemstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/community/posttype"
	"smart-wardrobe-be/internal/shared/domain/constants/community/transferstate"
	"smart-wardrobe-be/internal/shared/domain/constants/shared/requeststatus"
	"smart-wardrobe-be/internal/shared/domain/constants/wardrobe/itemcondition"

	"github.com/google/uuid"
)

type PostMediaReq struct {
	MediaType string  `json:"mediaType" binding:"required" label:"loại phương tiện"`
	MediaURL  string  `json:"mediaUrl" binding:"required,url" label:"đường dẫn phương tiện"`
	PublicID  *string `json:"publicId"`
	SortOrder int16   `json:"sortOrder"`
}

type PostItemInputReq struct {
	ItemID        uuid.UUID                    `json:"itemId" binding:"required" label:"trang phục"`
	Price         *float64                     `json:"price,omitempty"`
	ItemCondition *itemcondition.ItemCondition `json:"itemCondition,omitempty"`
}

type CreatePostReq struct {
	PostType    posttype.PostType  `json:"postType" binding:"required,oneof=OUTFIT SALE" label:"loại bài đăng"`
	Title       *string            `json:"title"`
	Content     string             `json:"content" binding:"required" label:"nội dung"`
	ContactInfo *string            `json:"contactInfo"`
	Items       []PostItemInputReq `json:"items"`
	Media       []PostMediaReq     `json:"media"`
}

type UpdatePostReq struct {
	Title       *string            `json:"title"`
	Content     string             `json:"content" binding:"required" label:"nội dung"`
	ContactInfo *string            `json:"contactInfo"`
	Items       []PostItemInputReq `json:"items"`
	Media       []PostMediaReq     `json:"media"`
}

type MarkPostItemsSoldReq struct {
	BuyerID     uuid.UUID   `json:"buyerId" binding:"required" label:"người mua"`
	PostItemIDs []uuid.UUID `json:"postItemIds" binding:"required,min=1" label:"danh sách trang phục"`
}

type CreateTransferRequestsReq struct {
	PostItemIDs []uuid.UUID `json:"postItemIds" binding:"required,min=1" label:"danh sách trang phục"`
}

type AcceptTransfersReq struct {
	PostItemIDs []uuid.UUID `json:"postItemIds" binding:"required,min=1" label:"danh sách trang phục"`
}

type TransferRequestRes struct {
	ID        uuid.UUID                   `json:"id"`
	BuyerID   uuid.UUID                   `json:"buyerId"`
	Username  string                      `json:"username"`
	AvatarURL *string                     `json:"avatarUrl,omitempty"`
	Status    requeststatus.RequestStatus `json:"status"`
	CreatedAt time.Time                   `json:"createdAt"`
}

type RemovePostItemsReq struct {
	PostItemIDs []uuid.UUID `json:"postItemIds" binding:"required" label:"danh sách món đồ bài đăng"`
}

type AddCommentReq struct {
	Content         string     `json:"content" binding:"required" label:"nội dung bình luận"`
	ParentCommentID *uuid.UUID `json:"parentCommentId,omitempty"`
}

type UpdateCommentReq struct {
	Content string `json:"content" binding:"required" label:"nội dung bình luận"`
}

type LikePostReq struct {
	IsLiked *bool `json:"isLiked" binding:"required" label:"trạng thái yêu thích"`
}

type GetFeedQueryReq struct {
	shared_dto.PaginationQuery
	Sort     string            `form:"sort"`
	Username string            `form:"username"`
	PostType posttype.PostType `form:"postType" binding:"omitempty,oneof=OUTFIT SALE" label:"loại bài đăng"`
}

type PostRes struct {
	ID                 uuid.UUID         `json:"id"`
	PublicID           string            `json:"publicId"`
	UserID             uuid.UUID         `json:"userId"`
	Username           string            `json:"username"`
	FirstName          string            `json:"firstName,omitempty"`
	LastName           string            `json:"lastName,omitempty"`
	AvatarURL          *string           `json:"avatarUrl,omitempty"`
	PostType           posttype.PostType `json:"postType"`
	Title              *string           `json:"title"`
	Content            string            `json:"content"`
	ContactInfo        *string           `json:"contactInfo"`
	TotalPrice         float64           `json:"totalPrice"`
	LikeCount          int               `json:"likeCount"`
	CommentCount       int               `json:"commentCount"`
	IsLiked            bool              `json:"isLiked"`
	GlobalHotnessScore float64           `json:"globalHotnessScore"`
	FinalFeedScore     float64           `json:"finalFeedScore,omitempty"`
	SharePath          string            `json:"sharePath"`
	Items              []*PostItemRes    `json:"items,omitempty"`
	Media              []*PostMediaRes   `json:"media,omitempty"`
	CreatedAt          time.Time         `json:"createdAt"`
	UpdatedAt          time.Time         `json:"updatedAt"`
	IsDeleted          bool              `json:"isDeleted,omitempty"`
}

type PostItemRes struct {
	ID            uuid.UUID                     `json:"id"`
	Item          *wardrobe_dto.WardrobeItemRes `json:"item"`
	Price         float64                       `json:"price"`
	ItemCondition itemcondition.ItemCondition   `json:"itemCondition"`
	Status        postitemstatus.PostItemStatus `json:"status"`
	BuyerUserID   *uuid.UUID                    `json:"buyerUserId"`
	TransferState transferstate.TransferState   `json:"transferState"`
	SoldAt        *time.Time                    `json:"soldAt"`
	DeclinedAt    *time.Time                    `json:"declinedAt,omitempty"`
}

type PostMediaRes struct {
	ID        uuid.UUID `json:"id"`
	MediaType string    `json:"mediaType"`
	MediaURL  string    `json:"mediaUrl"`
	PublicID  *string   `json:"publicId"`
	SortOrder int16     `json:"sortOrder"`
}

type CommentRes struct {
	ID              uuid.UUID  `json:"id"`
	UserID          uuid.UUID  `json:"userId"`
	Username        string     `json:"username"`
	FirstName       string     `json:"firstName,omitempty"`
	LastName        string     `json:"lastName,omitempty"`
	AvatarURL       *string    `json:"avatarUrl,omitempty"`
	Content         string     `json:"content"`
	ParentCommentID *uuid.UUID `json:"parentCommentId,omitempty"`
	CreatedAt       time.Time  `json:"createdAt"`
}

type PostLikeUserRes struct {
	ID        uuid.UUID `json:"id"`
	Username  string    `json:"username"`
	FirstName string    `json:"firstName,omitempty"`
	LastName  string    `json:"lastName,omitempty"`
	AvatarURL *string   `json:"avatarUrl,omitempty"`
}

type PendingTransferRes struct {
	PostItemID uuid.UUID                     `json:"postItemId"`
	Item       *wardrobe_dto.WardrobeItemRes `json:"item"`
	SellerName string                        `json:"sellerName"`
}

type TransferBuyerSummaryRes struct {
	ID        uuid.UUID `json:"id"`
	Username  string    `json:"username"`
	AvatarURL *string   `json:"avatarUrl,omitempty"`
}

type SellerTransferPostItemRes struct {
	PostItemID    uuid.UUID                     `json:"postItemId"`
	Item          *wardrobe_dto.WardrobeItemRes `json:"item"`
	Price         float64                       `json:"price"`
	ItemCondition itemcondition.ItemCondition   `json:"itemCondition"`
	Status        postitemstatus.PostItemStatus `json:"status"`
	TransferState transferstate.TransferState   `json:"transferState"`
	SoldAt        *time.Time                    `json:"soldAt,omitempty"`
	DeclinedAt    *time.Time                    `json:"declinedAt,omitempty"`
	Buyer         *TransferBuyerSummaryRes      `json:"buyer,omitempty"`
}

type SellerTransferPostRes struct {
	PostID    uuid.UUID                    `json:"postId"`
	Title     *string                      `json:"title"`
	PostType  posttype.PostType            `json:"postType"`
	CreatedAt time.Time                    `json:"createdAt"`
	UpdatedAt time.Time                    `json:"updatedAt"`
	Items     []*SellerTransferPostItemRes `json:"items"`
}

func NewTransferBuyerSummary(user *identity_dto.UserRes) *TransferBuyerSummaryRes {
	if user == nil {
		return nil
	}

	return &TransferBuyerSummaryRes{
		ID:        user.ID,
		Username:  user.Username,
		AvatarURL: user.AvatarUrl,
	}
}

type UploadSignatureResult = shared_dto.UploadSignatureResult
type GetFeedRes = shared_dto.PaginationResult[*PostRes]
