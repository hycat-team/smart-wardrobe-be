package persistence

import (
	"context"
	"errors"

	"smart-wardrobe-be/internal/modules/dashboard/domain/entities"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (r *DashboardRepository) UpsertUserWardrobeSummary(ctx context.Context, summary *entities.DashboardUserWardrobeSummary) error {
	return r.getDB(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"total_items", "total_wardrobe_value_vnd", "average_cost_per_wear_vnd",
			"underutilized_items_count", "underutilized_items",
			"ai_quota_remaining", "expiring_vouchers_count", "updated_at",
		}),
	}).Create(summary).Error
}

// GetWardrobeCategoryDistribution returns item count grouped by category for a user
func (r *DashboardRepository) GetWardrobeCategoryDistribution(ctx context.Context, userID uuid.UUID) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	err := r.getDB(ctx).Table("wardrobe_items").
		Select("c.id as category_id, c.name as category_name, COUNT(*) as item_count").
		Joins("JOIN fashion_items fi ON fi.id = wardrobe_items.fashion_item_id").
		Joins("JOIN categories c ON c.id = fi.category_id").
		Where("wardrobe_items.user_id = ? AND wardrobe_items.is_deleted = false", userID).
		Group("c.id, c.name").
		Order("item_count DESC").
		Scan(&results).Error
	if err != nil {
		return nil, err
	}
	return results, nil
}

func (r *DashboardRepository) GetUserWardrobeSummary(ctx context.Context, userID uuid.UUID) (*entities.DashboardUserWardrobeSummary, error) {
	var summary entities.DashboardUserWardrobeSummary
	err := r.getDB(ctx).
		Where("user_id = ?", userID).
		First(&summary).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &summary, nil
}
