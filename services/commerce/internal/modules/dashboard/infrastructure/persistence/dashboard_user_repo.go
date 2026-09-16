package persistence

import (
	"context"
	"errors"

	"smart-wardrobe-be/services/commerce/internal/modules/dashboard/domain/entities"

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
