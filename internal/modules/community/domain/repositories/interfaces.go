package repositories

import (
	"context"

	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"

	"github.com/google/uuid"
)

type IPostRepository interface {
	shared_repos.IGenericRepository[entities.Post, uuid.UUID]
	GetFeed(ctx context.Context) ([]*entities.Post, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*entities.Post, error)
	GetDetail(ctx context.Context, postID uuid.UUID) (*entities.Post, []*entities.PostItem, []*entities.PostMedia, error)
}

type IPostItemRepository interface {
	shared_repos.IGenericRepository[entities.PostItem, uuid.UUID]
	GetByPostID(ctx context.Context, postID uuid.UUID) ([]*entities.PostItem, error)
	GetPendingByBuyerID(ctx context.Context, buyerUserID uuid.UUID) ([]*entities.PostItem, error)
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
}

type ILikeRepository interface {
	shared_repos.IGenericRepository[entities.Like, uuid.UUID]
	GetPostLike(ctx context.Context, userID uuid.UUID, postID uuid.UUID) (*entities.Like, error)
}
