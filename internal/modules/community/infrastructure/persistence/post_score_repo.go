package persistence

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/modules/community/domain/dto"
	"smart-wardrobe-be/internal/modules/community/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/entities"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PostScoreRepository struct {
	db *gorm.DB
}

func NewPostScoreRepository(db *gorm.DB) repositories.IPostScoreRepository {
	return &PostScoreRepository{db: db}
}

func (r *PostScoreRepository) getDB(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx)
}

func (r *PostScoreRepository) GetScoresByPostIDs(ctx context.Context, postIDs []uuid.UUID) (map[uuid.UUID]float64, error) {
	result := make(map[uuid.UUID]float64, len(postIDs))
	if len(postIDs) == 0 {
		return result, nil
	}

	var rows []*entities.PostScoreSnapshot
	if err := r.getDB(ctx).Where("post_id IN ?", postIDs).Find(&rows).Error; err != nil {
		return nil, err
	}

	for _, row := range rows {
		result[row.PostID] = row.GlobalHotnessScore
	}
	return result, nil
}

func (r *PostScoreRepository) ListScoreMetricsByPostIDs(
	ctx context.Context, postIDs []uuid.UUID,
) ([]*dto.PostScoreMetric, error) {
	if len(postIDs) == 0 {
		return []*dto.PostScoreMetric{}, nil
	}

	var rows []struct {
		PostID        uuid.UUID `gorm:"column:post_id"`
		LikeCount     int       `gorm:"column:like_count"`
		CommentCount  int       `gorm:"column:comment_count"`
		CreatedAtTime time.Time `gorm:"column:created_at"`
	}

	if err := r.getDB(ctx).
		Model(&entities.Post{}).
		Select("id AS post_id, like_count, comment_count, created_at").
		Where("id IN ?", postIDs).
		Find(&rows).Error; err != nil {
		return nil, err
	}

	result := make([]*dto.PostScoreMetric, 0, len(rows))
	for _, row := range rows {
		result = append(result, &dto.PostScoreMetric{
			PostID:        row.PostID,
			LikeCount:     row.LikeCount,
			CommentCount:  row.CommentCount,
			CreatedAtUnix: row.CreatedAtTime.Unix(),
		})
	}

	return result, nil
}

func (r *PostScoreRepository) UpsertScores(ctx context.Context, items []*entities.PostScoreSnapshot) error {
	if len(items) == 0 {
		return nil
	}

	return r.getDB(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "post_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"global_hotness_score": gorm.Expr("EXCLUDED.global_hotness_score"),
			"last_calculated_at":   gorm.Expr("EXCLUDED.last_calculated_at"),
		}),
	}).Create(&items).Error
}
