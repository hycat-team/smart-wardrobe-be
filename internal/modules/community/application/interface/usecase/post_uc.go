package usecase

import (
	"context"

	"smart-wardrobe-be/internal/modules/community/application/dto"

	"github.com/google/uuid"
)

type IPostUseCase interface {
	ICommunityFeedUseCase
	IPostPublishingUseCase
}

type IUserPostUseCase interface {
	ICommunityFeedUseCase
	IUserPostManagementUseCase
}

type IAdminCommunityModerationUseCase interface {
	AdminDeletePost(ctx context.Context, adminUserID uuid.UUID, postPublicID string) error
	AdminHidePostItem(ctx context.Context, adminUserID uuid.UUID, postItemID uuid.UUID) error
	AdminDeletePostItem(ctx context.Context, adminUserID uuid.UUID, postItemID uuid.UUID) error
	AdminDeleteComment(ctx context.Context, adminUserID uuid.UUID, commentID uuid.UUID) error
	GetPostsForAdmin(ctx context.Context, query dto.AdminGetPostsQueryReq) (*dto.AdminPostListRes, error)
	GetPostItemsForAdmin(ctx context.Context, query dto.AdminGetPostItemsQueryReq) (*dto.AdminPostItemListRes, error)
	AdminRestorePost(ctx context.Context, adminUserID uuid.UUID, postPublicID string) error
	AdminRestoreComment(ctx context.Context, adminUserID uuid.UUID, commentID uuid.UUID) error
}

type ICommunityFeedUseCase interface {
	GetFeed(ctx context.Context, viewerUserID *uuid.UUID, query dto.GetFeedQueryReq) (*dto.GetFeedRes, error)
	GetPostDetail(ctx context.Context, postPublicID string, viewerUserID *uuid.UUID) (*dto.PostRes, error)
	GetPostComments(ctx context.Context, postPublicID string) ([]*dto.CommentRes, error)
	GetCommentReplies(ctx context.Context, postPublicID string, commentID uuid.UUID) ([]*dto.CommentRes, error)
	GetPostLikes(ctx context.Context, postPublicID string) ([]*dto.PostLikeUserRes, error)
}

type IPostPublishingUseCase interface {
	CreatePost(ctx context.Context, userID uuid.UUID, input dto.CreatePostReq) (*dto.PostRes, error)
	UpdatePost(ctx context.Context, userID uuid.UUID, postPublicID string, input dto.UpdatePostReq) (*dto.PostRes, error)
	DeletePost(ctx context.Context, userID uuid.UUID, postPublicID string) error
	RemovePostItems(ctx context.Context, userID uuid.UUID, postPublicID string, postItemIDs []uuid.UUID) error
	GetUploadSignature(ctx context.Context) (*dto.UploadSignatureResult, error)
}

type IUserPostManagementUseCase interface {
	CreatePost(ctx context.Context, userID uuid.UUID, input dto.CreatePostReq) (*dto.PostRes, error)
	UpdatePost(ctx context.Context, userID uuid.UUID, postPublicID string, input dto.UpdatePostReq) (*dto.PostRes, error)
	DeletePost(ctx context.Context, userID uuid.UUID, postPublicID string) error
	RemovePostItems(ctx context.Context, userID uuid.UUID, postPublicID string, postItemIDs []uuid.UUID) error
	GetUploadSignature(ctx context.Context) (*dto.UploadSignatureResult, error)
}
