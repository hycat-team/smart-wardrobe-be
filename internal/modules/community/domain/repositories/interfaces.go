package repositories

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/modules/community/domain/dto"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"

	"github.com/google/uuid"
)

type AdminPostFilter struct {
	PostType  *string
	IsDeleted *bool
	Query     *string
	Page      int
	Limit     int
}

type AdminPostListResult struct {
	Posts      []*entities.Post
	TotalCount int64
}

type IPostRepository interface {
	shared_repos.IGenericRepository[entities.Post, uuid.UUID]
	GetFeed(ctx context.Context, query dto.FeedQuery) (*dto.FeedResult, error)
	GetHotFeedCandidates(ctx context.Context, query dto.FeedQuery, limit int) ([]*dto.FeedPostRecord, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*entities.Post, error)
	GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*entities.Post, error)
	GetByPublicID(ctx context.Context, publicID string) (*entities.Post, error)
	GetDetail(ctx context.Context, postPublicID string) (*entities.Post, []*entities.PostItem, []*entities.PostMedia, error)
	GetDirtyPostIDs(ctx context.Context, limit int) ([]uuid.UUID, error)
	GetDecayRefreshPostIDs(ctx context.Context, since time.Time, limit int) ([]uuid.UUID, error)
	GetHighScoreStalePostIDs(ctx context.Context, before time.Time, minScore float64, limit int) ([]uuid.UUID, error)
	MarkHotnessDirty(ctx context.Context, postID uuid.UUID) error
	ClearHotnessDirty(ctx context.Context, postIDs []uuid.UUID) error
	SoftDelete(ctx context.Context, postID uuid.UUID) error
	Restore(ctx context.Context, postID uuid.UUID) error
	GetPostsForAdmin(ctx context.Context, filter AdminPostFilter) (*AdminPostListResult, error)
}

type IPostScoreRepository interface {
	GetScoresByPostIDs(ctx context.Context, postIDs []uuid.UUID) (map[uuid.UUID]float64, error)
	ListScoreMetricsByPostIDs(ctx context.Context, postIDs []uuid.UUID) ([]*dto.PostScoreMetric, error)
	UpsertScores(ctx context.Context, items []*entities.PostScoreSnapshot) error
}

type AdminPostItemFilter struct {
	Status        *int
	TransferState *int
	Page          int
	Limit         int
}

type AdminPostItemListResult struct {
	PostItems  []*entities.PostItem
	TotalCount int64
}

type IPostItemRepository interface {
	shared_repos.IGenericRepository[entities.PostItem, uuid.UUID]
	GetByPostID(ctx context.Context, postID uuid.UUID) ([]*entities.PostItem, error)
	GetByPostIDs(ctx context.Context, postIDs []uuid.UUID) ([]*entities.PostItem, error)
	GetPendingByBuyerID(ctx context.Context, buyerUserID uuid.UUID) ([]*entities.PostItem, error)
	GetTransferItemsBySellerID(ctx context.Context, sellerUserID uuid.UUID) ([]*entities.PostItem, error)
	GetByItemID(ctx context.Context, itemID uuid.UUID) ([]*entities.PostItem, error)
	GetSiblingItems(ctx context.Context, itemID uuid.UUID, excludePostItemID uuid.UUID) ([]*entities.PostItem, error)
	HasActiveTransfer(ctx context.Context, itemID uuid.UUID, excludePostItemID *uuid.UUID) (bool, error)
	GetActiveTransfersByItemIDs(ctx context.Context, itemIDs []uuid.UUID) ([]*entities.PostItem, error)
	DeleteByPostAndIDs(ctx context.Context, postID uuid.UUID, ids []uuid.UUID) error
	SumVisiblePriceByPostID(ctx context.Context, postID uuid.UUID) (float64, error)
	GetPostItemsForAdmin(ctx context.Context, filter AdminPostItemFilter) (*AdminPostItemListResult, error)
	GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*entities.PostItem, error)
	GetSiblingItemsByItemIDs(ctx context.Context, itemIDs []uuid.UUID, excludePostItemIDs []uuid.UUID) ([]*entities.PostItem, error)
}

type IPostMediaRepository interface {
	shared_repos.IGenericRepository[entities.PostMedia, uuid.UUID]
	GetByPostID(ctx context.Context, postID uuid.UUID) ([]*entities.PostMedia, error)
	GetByPostIDs(ctx context.Context, postIDs []uuid.UUID) ([]*entities.PostMedia, error)
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
	Restore(ctx context.Context, commentID uuid.UUID) error
}

type ILikeRepository interface {
	shared_repos.IGenericRepository[entities.Like, uuid.UUID]
	GetPostLike(ctx context.Context, userID uuid.UUID, postID uuid.UUID) (*entities.Like, error)
	GetLikedPostIDs(ctx context.Context, userID uuid.UUID, postIDs []uuid.UUID) (map[uuid.UUID]bool, error)
	GetUsersByPostID(ctx context.Context, postID uuid.UUID) ([]*entities.User, error)
}

type ITransferRequestRepository interface {
	shared_repos.IGenericRepository[entities.TransferRequest, uuid.UUID]
	GetByPostItemID(ctx context.Context, postItemID uuid.UUID) ([]*entities.TransferRequest, error)
	GetPendingByBuyerAndItems(ctx context.Context, buyerID uuid.UUID, postItemIDs []uuid.UUID) ([]*entities.TransferRequest, error)
	GetByBuyerAndPostItem(ctx context.Context, buyerID uuid.UUID, postItemID uuid.UUID) (*entities.TransferRequest, error)
	GetByBuyerAndPostItems(ctx context.Context, buyerID uuid.UUID, postItemIDs []uuid.UUID) ([]*entities.TransferRequest, error)
	GetByPostItemIDs(ctx context.Context, postItemIDs []uuid.UUID) ([]*entities.TransferRequest, error)
}
