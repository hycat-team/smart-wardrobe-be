package dto

import (
	"time"

	shared_dto "smart-wardrobe-be/shared/application/dto"

	"github.com/google/uuid"
)

type AddCommentReq struct {
	Content         string     `json:"content" binding:"required" label:"nội dung bình luận"`
	ParentCommentID *uuid.UUID `json:"parentCommentId" label:"bình luận cha"`
}

type UpdateCommentReq struct {
	Content string `json:"content" binding:"required" label:"nội dung bình luận"`
}

type LikePostReq struct {
	IsLiked *bool `json:"isLiked" binding:"required" label:"trạng thái yêu thích"`
}

type CommentRes struct {
	ID              uuid.UUID         `json:"id"`
	User            *CommunityUserRes `json:"user"`
	Content         string            `json:"content"`
	ParentCommentID *uuid.UUID        `json:"parentCommentId,omitempty"`
	ReplyCount      int64             `json:"replyCount"`
	IsDeleted       bool              `json:"isDeleted"`
	CreatedAt       time.Time         `json:"createdAt"`
}

type GetLikesQueryReq struct {
	shared_dto.PaginationQuery
}

type PostLikeUserListRes = shared_dto.PaginationResult[*CommunityUserRes]
