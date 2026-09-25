package repositories

import (
	"context"

	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/constants/hiddenreason"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/constants/poststatus"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/entities"
	shared_dto "smart-wardrobe-be/shared/application/dto"
	shared_repos "smart-wardrobe-be/shared/domain/repositories"

	"github.com/google/uuid"
)

type IPostRepository interface {
	shared_repos.IGenericRepository[entities.Post, uuid.UUID]
	GetByPublicID(ctx context.Context, publicID string) (*entities.Post, error)
	ListFeed(ctx context.Context, filter PostFilter) (*PostListResult, error)
	Search(ctx context.Context, filter SearchPostsFilter) (*PostListResult, error)
	AdminList(ctx context.Context, filter AdminPostFilter) (*PostListResult, error)
	ListByAuthorPaginated(ctx context.Context, authorID uuid.UUID, viewerUserID *uuid.UUID, pagination shared_dto.PaginationQuery) (*PostListResult, error)
	IncrementLikeCount(ctx context.Context, postID uuid.UUID, delta int) error
	IncrementCommentCount(ctx context.Context, postID uuid.UUID, delta int) error
	UpdateModerationStatus(ctx context.Context, postID uuid.UUID, status poststatus.PostStatus, moderatedBy *uuid.UUID, hiddenReason *hiddenreason.HiddenReason) error
	SoftDeleteByOutfitID(ctx context.Context, outfitID uuid.UUID) (int64, error)
	HideAllByAuthor(ctx context.Context, authorID uuid.UUID) (int64, error)
	RestoreAllByAuthor(ctx context.Context, authorID uuid.UUID) (int64, error)
	CountByAuthor(ctx context.Context, authorID uuid.UUID) (int64, error)
}

type IPostMediaRepository interface {
	BulkCreate(ctx context.Context, medias []*entities.PostMedia) error
	DeleteByPostID(ctx context.Context, postID uuid.UUID) error
	GetByPostIDs(ctx context.Context, postIDs []uuid.UUID) ([]*entities.PostMedia, error)
}

type IPostLikeRepository interface {
	Create(ctx context.Context, like *entities.PostLike) error
	Delete(ctx context.Context, postID, userID uuid.UUID) error
	Exists(ctx context.Context, postID, userID uuid.UUID) (bool, error)
	GetLikedPostIDs(ctx context.Context, userID uuid.UUID, postIDs []uuid.UUID) (map[uuid.UUID]bool, error)
	GetByPostIDPaginated(ctx context.Context, postID uuid.UUID, pagination shared_dto.PaginationQuery) ([]*entities.PostLike, int64, error)
}

type IPostCommentRepository interface {
	shared_repos.IGenericRepository[entities.PostComment, uuid.UUID]
	GetRoots(ctx context.Context, postID uuid.UUID) ([]*entities.PostComment, error)
	GetReplies(ctx context.Context, rootID uuid.UUID) ([]*entities.PostComment, error)
	CountRepliesByRootIDs(ctx context.Context, rootIDs []uuid.UUID) (map[uuid.UUID]int64, error)
	AdminList(ctx context.Context, filter AdminCommentFilter) (*CommentListResult, error)
	UpdateStatusByRoot(ctx context.Context, rootID uuid.UUID, status poststatus.PostStatus, moderatedBy *uuid.UUID) (int64, error)
}

type IFollowRepository interface {
	Create(ctx context.Context, follow *entities.Follow) error
	Delete(ctx context.Context, followerID, followeeID uuid.UUID) error
	Exists(ctx context.Context, followerID, followeeID uuid.UUID) (bool, error)
	GetFolloweeIDs(ctx context.Context, followerID uuid.UUID) ([]uuid.UUID, error)
	ListFollowing(ctx context.Context, followerID uuid.UUID) ([]FollowEdge, error)
	ListFollowers(ctx context.Context, followeeID uuid.UUID) ([]FollowEdge, error)
	CountFollowing(ctx context.Context, followerID uuid.UUID) (int64, error)
	CountFollowers(ctx context.Context, followeeID uuid.UUID) (int64, error)
}

type IUserFollowStatsRepository interface {
	Increment(ctx context.Context, userID uuid.UUID, followerDelta, followingDelta int) error
	GetByUserID(ctx context.Context, userID uuid.UUID) (*entities.UserFollowStats, error)
	GetByUserIDs(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]*entities.UserFollowStats, error)
	Reconcile(ctx context.Context) error
}
