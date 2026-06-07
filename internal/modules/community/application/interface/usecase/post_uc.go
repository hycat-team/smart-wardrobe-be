package usecase

import (
	"context"

	"smart-wardrobe-be/internal/modules/community/application/dto"

	"github.com/google/uuid"
)

type IPostUseCase interface {
	IPostFeedUseCase
	IPostWriteUseCase
	IPostAssetUseCase
}

type IUserPostUseCase interface {
	IPostFeedUseCase
	IUserPostWriteUseCase
	IPostAssetUseCase
}

type IAdminCommunityModerationUseCase interface {
	AdminDeletePost(ctx context.Context, adminUserID uuid.UUID, postID uuid.UUID) error
	AdminHidePostItem(ctx context.Context, adminUserID uuid.UUID, postItemID uuid.UUID) error
	AdminDeletePostItem(ctx context.Context, adminUserID uuid.UUID, postItemID uuid.UUID) error
	AdminDeleteComment(ctx context.Context, adminUserID uuid.UUID, commentID uuid.UUID) error
}

type IPostFeedUseCase interface {
	GetFeed(ctx context.Context, viewerUserID *uuid.UUID, query dto.GetFeedQueryReq) (*dto.GetFeedRes, error)
	GetPostDetail(ctx context.Context, postID uuid.UUID, viewerUserID *uuid.UUID) (*dto.PostRes, error)
}

type IPostWriteUseCase interface {
	CreatePost(ctx context.Context, userID uuid.UUID, input dto.CreatePostReq) (*dto.PostRes, error)
	DeletePost(ctx context.Context, userID uuid.UUID, postID uuid.UUID) error
	RemovePostItems(ctx context.Context, userID uuid.UUID, postID uuid.UUID, postItemIDs []uuid.UUID) error
	AdminDeletePost(ctx context.Context, adminUserID uuid.UUID, postID uuid.UUID) error
	AdminHidePostItem(ctx context.Context, adminUserID uuid.UUID, postItemID uuid.UUID) error
	AdminDeletePostItem(ctx context.Context, adminUserID uuid.UUID, postItemID uuid.UUID) error
}

type IUserPostWriteUseCase interface {
	CreatePost(ctx context.Context, userID uuid.UUID, input dto.CreatePostReq) (*dto.PostRes, error)
	DeletePost(ctx context.Context, userID uuid.UUID, postID uuid.UUID) error
	RemovePostItems(ctx context.Context, userID uuid.UUID, postID uuid.UUID, postItemIDs []uuid.UUID) error
}

type IAdminPostModerationUseCase interface {
	AdminDeletePost(ctx context.Context, adminUserID uuid.UUID, postID uuid.UUID) error
	AdminHidePostItem(ctx context.Context, adminUserID uuid.UUID, postItemID uuid.UUID) error
	AdminDeletePostItem(ctx context.Context, adminUserID uuid.UUID, postItemID uuid.UUID) error
}

type IPostAssetUseCase interface {
	GetUploadSignature(ctx context.Context) (*dto.UploadSignatureResult, error)
}
