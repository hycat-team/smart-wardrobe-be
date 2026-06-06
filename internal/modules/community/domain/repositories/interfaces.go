package repositories

import (
	"context"

	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"

	"github.com/google/uuid"
)

type FeedQuery struct {
	Sort     string
	Page     int
	Limit    int
	UserID   *uuid.UUID
	PostType *string
}

type FeedResult struct {
	Items    []*FeedPostRecord
	Metadata shared_dto.PaginationMetadata
}

type FeedPostRecord struct {
	Post               *entities.Post
	GlobalHotnessScore float64
}

type PostScoreMetric struct {
	PostID        uuid.UUID
	LikeCount     int
	CommentCount  int
	CreatedAtUnix int64
}

type IPostRepository interface {
	shared_repos.IGenericRepository[entities.Post, uuid.UUID]
	GetFeed(ctx context.Context, query FeedQuery) (*FeedResult, error)
	GetHotFeedCandidates(ctx context.Context, query FeedQuery, limit int) ([]*FeedPostRecord, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*entities.Post, error)
	GetDetail(ctx context.Context, postID uuid.UUID) (*entities.Post, []*entities.PostItem, []*entities.PostMedia, error)
}

type IPostScoreRepository interface {
	GetScoresByPostIDs(ctx context.Context, postIDs []uuid.UUID) (map[uuid.UUID]float64, error)
	ListScoreMetrics(ctx context.Context) ([]*PostScoreMetric, error)
	UpsertScores(ctx context.Context, items []*entities.PostScoreSnapshot) error
}

type IPostItemRepository interface {
	shared_repos.IGenericRepository[entities.PostItem, uuid.UUID]
	GetByPostID(ctx context.Context, postID uuid.UUID) ([]*entities.PostItem, error)
	GetPendingByBuyerID(ctx context.Context, buyerUserID uuid.UUID) ([]*entities.PostItem, error)
	GetByItemID(ctx context.Context, itemID uuid.UUID) ([]*entities.PostItem, error)
	GetSiblingItems(ctx context.Context, itemID uuid.UUID, excludePostItemID uuid.UUID) ([]*entities.PostItem, error)
	HasActiveTransfer(ctx context.Context, itemID uuid.UUID, excludePostItemID *uuid.UUID) (bool, error)
	DeleteByPostAndIDs(ctx context.Context, postID uuid.UUID, ids []uuid.UUID) error
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
	GetByIDAndPostID(ctx context.Context, commentID uuid.UUID, postID uuid.UUID) (*entities.Comment, error)
}

type ILikeRepository interface {
	shared_repos.IGenericRepository[entities.Like, uuid.UUID]
	GetPostLike(ctx context.Context, userID uuid.UUID, postID uuid.UUID) (*entities.Like, error)
	GetLikedPostIDs(ctx context.Context, userID uuid.UUID, postIDs []uuid.UUID) (map[uuid.UUID]bool, error)
}
