package persistence

import (
	"context"

	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/entities"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserFollowStatsRepository struct {
	db *gorm.DB
}

func NewUserFollowStatsRepository(db *gorm.DB) repositories.IUserFollowStatsRepository {
	return &UserFollowStatsRepository{db: db}
}

// Increment upserts a stats row and applies deltas atomically, clamping at zero.
func (r *UserFollowStatsRepository) Increment(ctx context.Context, userID uuid.UUID, followerDelta, followingDelta int) error {
	return getDB(ctx, r.db).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"follower_count":  gorm.Expr("GREATEST(user_follow_stats.follower_count + ?, 0)", followerDelta),
			"following_count": gorm.Expr("GREATEST(user_follow_stats.following_count + ?, 0)", followingDelta),
			"updated_at":      gorm.Expr("now()"),
		}),
	}).Create(&entities.UserFollowStats{
		UserID:         userID,
		FollowerCount:  maxInt(followerDelta, 0),
		FollowingCount: maxInt(followingDelta, 0),
	}).Error
}

func (r *UserFollowStatsRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*entities.UserFollowStats, error) {
	var stats entities.UserFollowStats
	err := getDB(ctx, r.db).Where("user_id = ?", userID).First(&stats).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &stats, nil
}

func (r *UserFollowStatsRepository) GetByUserIDs(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]*entities.UserFollowStats, error) {
	result := make(map[uuid.UUID]*entities.UserFollowStats, len(userIDs))
	if len(userIDs) == 0 {
		return result, nil
	}
	var rows []*entities.UserFollowStats
	if err := getDB(ctx, r.db).Where("user_id IN ?", userIDs).Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.UserID] = row
	}
	return result, nil
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Reconcile rebuilds follower/following counters from the follows table.
const reconcileFollowStatsSQL = `
WITH follower AS (
    SELECT followee_id AS user_id, COUNT(*) AS c FROM follows GROUP BY followee_id
), following AS (
    SELECT follower_id AS user_id, COUNT(*) AS c FROM follows GROUP BY follower_id
), ids AS (
    SELECT followee_id AS user_id FROM follows
    UNION
    SELECT follower_id AS user_id FROM follows
)
INSERT INTO user_follow_stats (user_id, follower_count, following_count, updated_at)
SELECT ids.user_id, COALESCE(follower.c, 0), COALESCE(following.c, 0), now()
FROM ids
LEFT JOIN follower ON follower.user_id = ids.user_id
LEFT JOIN following ON following.user_id = ids.user_id
ON CONFLICT (user_id) DO UPDATE
SET follower_count = EXCLUDED.follower_count,
    following_count = EXCLUDED.following_count,
    updated_at = now()`

func (r *UserFollowStatsRepository) Reconcile(ctx context.Context) error {
	return getDB(ctx, r.db).Exec(reconcileFollowStatsSQL).Error
}
