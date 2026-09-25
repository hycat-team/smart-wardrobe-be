package persistence

import (
	"context"
	"time"

	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/entities"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type FollowRepository struct {
	db *gorm.DB
}

func NewFollowRepository(db *gorm.DB) repositories.IFollowRepository {
	return &FollowRepository{db: db}
}

func (r *FollowRepository) Create(ctx context.Context, follow *entities.Follow) error {
	return getDB(ctx, r.db).Clauses(clause.OnConflict{DoNothing: true}).Create(follow).Error
}

func (r *FollowRepository) Delete(ctx context.Context, followerID, followeeID uuid.UUID) error {
	return getDB(ctx, r.db).
		Where("follower_id = ? AND followee_id = ?", followerID, followeeID).
		Delete(&entities.Follow{}).Error
}

func (r *FollowRepository) Exists(ctx context.Context, followerID, followeeID uuid.UUID) (bool, error) {
	var count int64
	err := getDB(ctx, r.db).Model(&entities.Follow{}).
		Where("follower_id = ? AND followee_id = ?", followerID, followeeID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *FollowRepository) GetFolloweeIDs(ctx context.Context, followerID uuid.UUID) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := getDB(ctx, r.db).Model(&entities.Follow{}).
		Where("follower_id = ?", followerID).
		Pluck("followee_id", &ids).Error
	if err != nil {
		return nil, err
	}
	return ids, nil
}

func (r *FollowRepository) ListFollowing(ctx context.Context, followerID uuid.UUID) ([]repositories.FollowEdge, error) {
	return r.listEdges(ctx, "follower_id", "followee_id", followerID)
}

func (r *FollowRepository) ListFollowers(ctx context.Context, followeeID uuid.UUID) ([]repositories.FollowEdge, error) {
	return r.listEdges(ctx, "followee_id", "follower_id", followeeID)
}

func (r *FollowRepository) listEdges(ctx context.Context, filterColumn, selectColumn string, value uuid.UUID) ([]repositories.FollowEdge, error) {
	type row struct {
		UserID    uuid.UUID
		CreatedAt time.Time
	}
	var rows []row
	err := getDB(ctx, r.db).Model(&entities.Follow{}).
		Select(selectColumn+" AS user_id, created_at").
		Where(filterColumn+" = ?", value).
		Order("created_at DESC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	edges := make([]repositories.FollowEdge, 0, len(rows))
	for _, row := range rows {
		edges = append(edges, repositories.FollowEdge{UserID: row.UserID, FollowedAt: row.CreatedAt})
	}
	return edges, nil
}

func (r *FollowRepository) CountFollowing(ctx context.Context, followerID uuid.UUID) (int64, error) {
	var count int64
	err := getDB(ctx, r.db).Model(&entities.Follow{}).Where("follower_id = ?", followerID).Count(&count).Error
	return count, err
}

func (r *FollowRepository) CountFollowers(ctx context.Context, followeeID uuid.UUID) (int64, error) {
	var count int64
	err := getDB(ctx, r.db).Model(&entities.Follow{}).Where("followee_id = ?", followeeID).Count(&count).Error
	return count, err
}
