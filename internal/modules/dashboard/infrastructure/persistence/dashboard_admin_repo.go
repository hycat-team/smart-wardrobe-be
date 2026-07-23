package persistence

import (
	"context"
	"errors"
	"time"

	"smart-wardrobe-be/internal/modules/dashboard/domain/entities"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (r *DashboardRepository) UpsertAdminSnapshot(ctx context.Context, snapshot *entities.DashboardAdminDailySnapshot) error {
	return r.getDB(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "snapshot_date"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"total_users", "new_users", "active_users_dau", "total_brands",
			"pending_review_brands", "active_campaigns", "ai_requests_count",
			"ai_error_count", "ai_paid_tokens", "ai_actual_cost_micro_vnd",
			"subscription_revenue_vnd", "net_ai_margin_vnd", "payos_transaction_count",
			"payos_success_count", "updated_at",
		}),
	}).Create(snapshot).Error
}

func (r *DashboardRepository) GetAdminSnapshots(ctx context.Context, fromDate, toDate time.Time) ([]*entities.DashboardAdminDailySnapshot, error) {
	var snapshots []*entities.DashboardAdminDailySnapshot
	err := r.getDB(ctx).
		Where("snapshot_date >= ? AND snapshot_date <= ?", fromDate, toDate).
		Order("snapshot_date ASC").
		Find(&snapshots).Error
	if err != nil {
		return nil, err
	}
	return snapshots, nil
}

func (r *DashboardRepository) GetLatestAdminSnapshot(ctx context.Context) (*entities.DashboardAdminDailySnapshot, error) {
	var snapshot entities.DashboardAdminDailySnapshot
	err := r.getDB(ctx).
		Order("snapshot_date DESC").
		First(&snapshot).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &snapshot, nil
}
