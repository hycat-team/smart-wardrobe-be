package usecase

import (
	"context"

	"smart-wardrobe-be/internal/modules/community/application/dto"

	"github.com/google/uuid"
)

type IPostInteractionUseCase interface {
	TogglePostLike(ctx context.Context, userID uuid.UUID, postID uuid.UUID, isLiked bool) error
	AddComment(ctx context.Context, userID uuid.UUID, postID uuid.UUID, content string) (*dto.CommentRes, error)
}
