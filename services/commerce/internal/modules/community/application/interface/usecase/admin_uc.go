package usecase

import (
	"context"

	"smart-wardrobe-be/services/commerce/internal/modules/community/application/dto"

	"github.com/google/uuid"
)

type IAdminUseCase interface {
	ListPosts(ctx context.Context, query dto.GetAdminPostsQueryReq) (*dto.AdminPostListRes, error)
	HidePost(ctx context.Context, adminID, postID uuid.UUID) error
	RestorePost(ctx context.Context, adminID, postID uuid.UUID) error
	DeletePost(ctx context.Context, adminID, postID uuid.UUID) error
	ListComments(ctx context.Context, query dto.GetAdminCommentsQueryReq) (*dto.AdminCommentListRes, error)
	HideComment(ctx context.Context, adminID, commentID uuid.UUID) error
	RestoreComment(ctx context.Context, adminID, commentID uuid.UUID) error
	DeleteComment(ctx context.Context, adminID, commentID uuid.UUID) error
}
