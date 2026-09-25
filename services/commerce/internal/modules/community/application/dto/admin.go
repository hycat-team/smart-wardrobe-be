package dto

import (
	shared_dto "smart-wardrobe-be/shared/application/dto"

	"github.com/google/uuid"
)

type GetAdminPostsQueryReq struct {
	shared_dto.PaginationQuery
	Q      string `form:"q" label:"từ khóa"`
	Status string `form:"status" label:"trạng thái"`
}

type AdminPostRes struct {
	PostRes
	ModeratedBy  *uuid.UUID `json:"moderatedBy,omitempty"`
	HiddenReason *string    `json:"hiddenReason,omitempty"`
}

type AdminPostListRes = shared_dto.PaginationResult[*AdminPostRes]

type GetAdminCommentsQueryReq struct {
	shared_dto.PaginationQuery
	Q string `form:"q" label:"từ khóa"`
}

type AdminCommentRes struct {
	CommentRes
	PostID uuid.UUID `json:"postId"`
	Status string    `json:"status"`
}

type AdminCommentListRes = shared_dto.PaginationResult[*AdminCommentRes]
