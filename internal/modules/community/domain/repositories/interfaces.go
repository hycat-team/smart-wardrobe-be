package repositories

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/modules/community/domain/dto"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"

	"github.com/google/uuid"
)

type IPostRepository interface {
	shared_repos.IGenericRepository[entities.Post, uuid.UUID]
	GetFeed(ctx context.Context, query dto.FeedQuery) (*dto.FeedResult, error)
	GetHotFeedCandidates(ctx context.Context, query dto.FeedQuery, limit int) ([]*dto.FeedPostRecord, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*entities.Post, error)
	GetByPublicID(ctx context.Context, publicID string) (*entities.Post, error)
	GetDetail(ctx context.Context, postPublicID string) (*entities.Post, []*entities.PostItem, []*entities.PostMedia, error)
	GetDirtyPostIDs(ctx context.Context, limit int) ([]uuid.UUID, error)
	GetDecayRefreshPostIDs(ctx context.Context, since time.Time, limit int) ([]uuid.UUID, error)
	GetHighScoreStalePostIDs(ctx context.Context, before time.Time, minScore float64, limit int) ([]uuid.UUID, error)
	MarkHotnessDirty(ctx context.Context, postID uuid.UUID) error
	ClearHotnessDirty(ctx context.Context, postIDs []uuid.UUID) error
	SoftDelete(ctx context.Context, postID uuid.UUID) error
}

type IPostScoreRepository interface {
	GetScoresByPostIDs(ctx context.Context, postIDs []uuid.UUID) (map[uuid.UUID]float64, error)
	ListScoreMetricsByPostIDs(ctx context.Context, postIDs []uuid.UUID) ([]*dto.PostScoreMetric, error)
	UpsertScores(ctx context.Context, items []*entities.PostScoreSnapshot) error
}

type IPostItemRepository interface {
	shared_repos.IGenericRepository[entities.PostItem, uuid.UUID]
	GetByPostID(ctx context.Context, postID uuid.UUID) ([]*entities.PostItem, error)
	GetPendingByBuyerID(ctx context.Context, buyerUserID uuid.UUID) ([]*entities.PostItem, error)
	GetTransferItemsBySellerID(ctx context.Context, sellerUserID uuid.UUID) ([]*entities.PostItem, error)
	GetByItemID(ctx context.Context, itemID uuid.UUID) ([]*entities.PostItem, error)
	GetSiblingItems(ctx context.Context, itemID uuid.UUID, excludePostItemID uuid.UUID) ([]*entities.PostItem, error)
	HasActiveTransfer(ctx context.Context, itemID uuid.UUID, excludePostItemID *uuid.UUID) (bool, error)
	DeleteByPostAndIDs(ctx context.Context, postID uuid.UUID, ids []uuid.UUID) error
	SumVisiblePriceByPostID(ctx context.Context, postID uuid.UUID) (float64, error)
}

type IPostMediaRepository interface {
	shared_repos.IGenericRepository[entities.PostMedia, uuid.UUID]
	GetByPostID(ctx context.Context, postID uuid.UUID) ([]*entities.PostMedia, error)
	BulkCreate(ctx context.Context, items []*entities.PostMedia) error
	DeleteByPostID(ctx context.Context, postID uuid.UUID) error
}

type ICommentRepository interface {
	shared_repos.IGenericRepository[entities.Comment, uuid.UUID]
	GetByPostID(ctx context.Context, postID uuid.UUID) ([]*entities.Comment, error)
	GetTopLevelByPostID(ctx context.Context, postID uuid.UUID) ([]*entities.Comment, error)
	GetRepliesByParentID(ctx context.Context, postID uuid.UUID, parentCommentID uuid.UUID) ([]*entities.Comment, error)
	GetByIDAndPostID(ctx context.Context, commentID uuid.UUID, postID uuid.UUID) (*entities.Comment, error)
	SoftDelete(ctx context.Context, commentID uuid.UUID) error
	SoftDeleteByParentID(ctx context.Context, parentCommentID uuid.UUID) error
}

type ILikeRepository interface {
	shared_repos.IGenericRepository[entities.Like, uuid.UUID]
	GetPostLike(ctx context.Context, userID uuid.UUID, postID uuid.UUID) (*entities.Like, error)
	GetLikedPostIDs(ctx context.Context, userID uuid.UUID, postIDs []uuid.UUID) (map[uuid.UUID]bool, error)
	GetUsersByPostID(ctx context.Context, postID uuid.UUID) ([]*entities.User, error)
}
