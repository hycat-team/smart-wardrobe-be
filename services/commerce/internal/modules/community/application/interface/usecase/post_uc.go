package usecase

import (
	"context"

	"smart-wardrobe-be/services/commerce/internal/modules/community/application/dto"
	shared_dto "smart-wardrobe-be/shared/application/dto"

	"github.com/google/uuid"
)

type IPostUseCase interface {
	CreatePost(ctx context.Context, userID uuid.UUID, input dto.CreatePostReq) (*dto.PostRes, error)
	UpdatePost(ctx context.Context, userID uuid.UUID, publicID string, input dto.UpdatePostReq) (*dto.PostRes, error)
	DeletePost(ctx context.Context, userID uuid.UUID, publicID string) error
	GetPostDetail(ctx context.Context, publicID string, viewerUserID *uuid.UUID) (*dto.PostRes, error)
	GetFeed(ctx context.Context, viewerUserID *uuid.UUID, query dto.GetFeedQueryReq) (*dto.PostListRes, error)
	GetUserPosts(ctx context.Context, username string, viewerUserID *uuid.UUID, query dto.GetUserPostsQueryReq) (*dto.PostListRes, error)
	SearchPosts(ctx context.Context, query dto.SearchQueryReq, viewerUserID *uuid.UUID) (*dto.PostListRes, error)
	GetUploadSignature(ctx context.Context, resourceType string) (*shared_dto.UploadSignatureResult, error)
}
