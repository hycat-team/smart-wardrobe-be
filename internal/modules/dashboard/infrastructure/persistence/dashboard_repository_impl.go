package persistence

import (
	"context"
	"errors"
	"time"

	"smart-wardrobe-be/internal/modules/dashboard/domain/entities"
	"smart-wardrobe-be/internal/modules/dashboard/domain/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type DashboardRepository struct {
	db *gorm.DB
}

func NewDashboardRepository(db *gorm.DB) repositories.IDashboardRepository {
	return &DashboardRepository{db: db}
}

func (r *DashboardRepository) getDB(ctx context.Context) *gorm.DB {
	if ctx == nil {
		return r.db
	}
	if tx, ok := ctx.Value("TxKey").(*gorm.DB); ok && tx != nil {
		return tx.WithContext(ctx)
	}
	return r.db.WithContext(ctx)
}

func (r *DashboardRepository) UpsertAdminSnapshot(ctx context.Context, snapshot *entities.DashboardAdminDailySnapshot) error {
	return r.getDB(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "snapshot_date"}},
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

func (r *DashboardRepository) UpsertBrandSnapshot(ctx context.Context, snapshot *entities.DashboardBrandDailySnapshot) error {
	return r.getDB(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "brand_id"}, {Name: "snapshot_date"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"total_members", "points_issued", "points_redeemed", "points_expiring_30d",
			"points_liability_value_vnd", "active_vouchers", "vouchers_redeemed",
			"digital_sample_votes_count", "digital_sample_avg_rating", "open_cs_tickets",
			"cs_first_response_avg_seconds", "updated_at",
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

func (r *DashboardRepository) UpsertUserWardrobeSummary(ctx context.Context, summary *entities.DashboardUserWardrobeSummary) error {
	return r.getDB(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"total_items", "total_wardrobe_value_vnd", "average_cost_per_wear_vnd",
			"underutilized_items_count", "category_distribution", "color_distribution",
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
