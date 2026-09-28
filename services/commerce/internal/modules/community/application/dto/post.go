package dto

import (
	"time"

	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/constants/poststatus"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/constants/posttype"
	shared_dto "smart-wardrobe-be/shared/application/dto"

	"github.com/google/uuid"
)

type PostMediaReq struct {
	MediaType       string   `json:"mediaType" binding:"required,oneof=image video" label:"loại media"`
	MediaURL        string   `json:"mediaUrl" binding:"required,url" label:"đường dẫn media"`
	PublicID        *string  `json:"publicId" label:"mã media"`
	SortOrder       *int16   `json:"sortOrder" binding:"required" label:"thứ tự hiển thị"`
	Bytes           *int64   `json:"bytes" binding:"omitempty,gt=0" label:"kích thước media (byte)"`
	DurationSeconds *float64 `json:"durationSeconds" binding:"omitempty,gt=0" label:"thời lượng video (giây)"`
}

type CreatePostReq struct {
	PostType posttype.PostType `json:"postType" binding:"required,oneof=outfit media" label:"loại bài đăng"`
	Title    *string           `json:"title" label:"tiêu đề"`
	Content  string            `json:"content" label:"nội dung"`
	OutfitID *uuid.UUID        `json:"outfitId" label:"outfit"`
	Media    []PostMediaReq    `json:"media" label:"media"`
}

type UpdatePostReq struct {
	Title    *string        `json:"title" label:"tiêu đề"`
	Content  string         `json:"content" label:"nội dung"`
	OutfitID *uuid.UUID     `json:"outfitId" label:"outfit"`
	Media    []PostMediaReq `json:"media" label:"media"`
}

type GetFeedQueryReq struct {
	shared_dto.PaginationQuery
	Type     string `form:"type" label:"loại bảng tin"`
	Sort     string `form:"sort" label:"sắp xếp"`
	PostType string `form:"postType" label:"loại bài đăng"`
	Username string `form:"username" label:"tên người dùng"`
}

type PostMediaRes struct {
	ID        uuid.UUID `json:"id"`
	MediaType string    `json:"mediaType"`
	MediaURL  string    `json:"mediaUrl"`
	PublicID  *string   `json:"publicId,omitempty"`
	SortOrder int16     `json:"sortOrder"`
}

type OutfitBriefRes struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	CoverImageURL *string   `json:"coverImageUrl,omitempty"`
}

type PostRes struct {
	ID                uuid.UUID             `json:"id"`
	PublicID          string                `json:"publicId"`
	User              *CommunityUserRes     `json:"user"`
	PostType          posttype.PostType     `json:"postType"`
	Status            poststatus.PostStatus `json:"status"`
	Title             *string               `json:"title"`
	Content           string                `json:"content"`
	Outfit            *OutfitBriefRes       `json:"outfit,omitempty"`
	LikeCount         int                   `json:"likeCount"`
	CommentCount      int                   `json:"commentCount"`
	IsLiked           bool                  `json:"isLiked"`
	IsFollowingAuthor bool                  `json:"isFollowingAuthor"`
	SharePath         string                `json:"sharePath"`
	Media             []*PostMediaRes       `json:"media,omitempty"`
	CreatedAt         time.Time             `json:"createdAt"`
	UpdatedAt         time.Time             `json:"updatedAt"`
}

type PostListRes = shared_dto.PaginationResult[*PostRes]

type UploadSignatureResult = shared_dto.UploadSignatureResult
