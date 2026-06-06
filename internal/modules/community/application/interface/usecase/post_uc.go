package usecase

import (
	"context"

	"smart-wardrobe-be/internal/modules/community/application/dto"

	"github.com/google/uuid"
)

type IPostUseCase interface {
	CreatePost(ctx context.Context, userID uuid.UUID, input dto.CreatePostReq) (*dto.PostRes, error)
	GetFeed(ctx context.Context, viewerUserID *uuid.UUID, query dto.GetFeedQueryReq) (*dto.GetFeedRes, error)
	GetPostDetail(ctx context.Context, postID uuid.UUID, viewerUserID *uuid.UUID) (*dto.PostRes, error)
	GetUploadSignature(ctx context.Context) (*dto.UploadSignatureResult, error)
	DeletePost(ctx context.Context, userID uuid.UUID, postID uuid.UUID) error
	RemovePostItems(ctx context.Context, userID uuid.UUID, postID uuid.UUID, postItemIDs []uuid.UUID) error
}
