package persistence

import (
	"context"
	"errors"
	"time"

	"smart-wardrobe-be/services/commerce/internal/modules/dashboard/domain/entities"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (r *DashboardRepository) UpsertBrandSnapshot(ctx context.Context, snapshot *entities.DashboardBrandDailySnapshot) error {
	return r.getDB(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "brand_id"}, {Name: "snapshot_date"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"total_members", "points_issued", "points_redeemed", "points_expiring_30d",
			"points_liability_value_vnd", "active_vouchers", "vouchers_redeemed",
			"digital_sample_votes_count", "digital_sample_avg_rating", "open_cs_tickets",
			"updated_at",
		}),
	}).Create(snapshot).Error
}

func (r *DashboardRepository) GetBrandSnapshots(ctx context.Context, brandID uuid.UUID, fromDate, toDate time.Time) ([]*entities.DashboardBrandDailySnapshot, error) {
	var snapshots []*entities.DashboardBrandDailySnapshot
	err := r.getDB(ctx).
		Where("brand_id = ? AND snapshot_date >= ? AND snapshot_date <= ?", brandID, fromDate, toDate).
		Order("snapshot_date ASC").
		Find(&snapshots).Error
	if err != nil {
		return nil, err
	}
	return snapshots, nil
}

func (r *DashboardRepository) GetLatestBrandSnapshot(ctx context.Context, brandID uuid.UUID) (*entities.DashboardBrandDailySnapshot, error) {
	var snapshot entities.DashboardBrandDailySnapshot
	err := r.getDB(ctx).
		Where("brand_id = ?", brandID).
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
