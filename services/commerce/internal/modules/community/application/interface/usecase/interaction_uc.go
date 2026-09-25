package usecase

import (
	"context"

	"smart-wardrobe-be/services/commerce/internal/modules/community/application/dto"

	"github.com/google/uuid"
)

type IPostInteractionUseCase interface {
	TogglePostLike(ctx context.Context, userID uuid.UUID, publicID string, isLiked bool) error
	AddComment(ctx context.Context, userID uuid.UUID, publicID string, input dto.AddCommentReq) (*dto.CommentRes, error)
	UpdateComment(ctx context.Context, userID uuid.UUID, publicID string, commentID uuid.UUID, content string) (*dto.CommentRes, error)
	DeleteComment(ctx context.Context, userID uuid.UUID, publicID string, commentID uuid.UUID) error
	GetPostComments(ctx context.Context, publicID string) ([]*dto.CommentRes, error)
	GetCommentReplies(ctx context.Context, publicID string, rootCommentID uuid.UUID) ([]*dto.CommentRes, error)
	GetPostLikes(ctx context.Context, publicID string, query dto.GetLikesQueryReq) (*dto.PostLikeUserListRes, error)
}
