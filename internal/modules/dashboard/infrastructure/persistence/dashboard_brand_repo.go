package persistence

import (
	"context"
	"errors"
	"time"

	"smart-wardrobe-be/internal/modules/brand/domain/constants/loyaltypointlotstatus"
	"smart-wardrobe-be/internal/modules/dashboard/domain/entities"

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

// GetCustomerAcquisition returns customer count grouped by joined source
func (r *DashboardRepository) GetCustomerAcquisition(ctx context.Context, brandID uuid.UUID) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	err := r.getDB(ctx).Table("brand_customers").
		Select("joined_source as source, COUNT(*) as customer_count").
		Where("brand_id = ?", brandID).
		Group("joined_source").
		Order("customer_count DESC").
		Scan(&results).Error
	if err != nil {
		return nil, err
	}
	return results, nil
}

// GetLoyaltyTierDistribution returns member count grouped by loyalty tier
func (r *DashboardRepository) GetLoyaltyTierDistribution(ctx context.Context, brandID uuid.UUID) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	err := r.getDB(ctx).Table("loyalty_accounts").
		Select("lt.id as tier_id, lt.name as tier_name, lt.rank as tier_rank, COUNT(*) as member_count").
		Joins("JOIN loyalty_tiers lt ON lt.id = loyalty_accounts.current_tier_id").
		Where("loyalty_accounts.brand_id = ? AND loyalty_accounts.current_tier_id IS NOT NULL", brandID).
		Group("lt.id, lt.name, lt.rank").
		Order("lt.rank ASC").
		Scan(&results).Error
	if err != nil {
		return nil, err
	}
	return results, nil
}

// GetPointExpiryForecast returns points expiring per month for the next months
func (r *DashboardRepository) GetPointExpiryForecast(ctx context.Context, brandID uuid.UUID) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	err := r.getDB(ctx).Table("loyalty_point_lots").
		Select("DATE_TRUNC('month', expires_at)::date as month, COALESCE(SUM(remaining_points), 0) as points_expiring").
		Where("brand_id = ? AND status = ? AND remaining_points > 0 AND expires_at IS NOT NULL", brandID, string(loyaltypointlotstatus.Active)).
		Group("DATE_TRUNC('month', expires_at)").
		Order("month ASC").
		Limit(6).
		Scan(&results).Error
	if err != nil {
		return nil, err
	}
	return results, nil
}

// GetBenefitRedemptionAnalytics returns top benefits by redemption count and monthly trend
func (r *DashboardRepository) GetBenefitRedemptionAnalytics(ctx context.Context, brandID uuid.UUID) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	err := r.getDB(ctx).Table("benefit_redemptions").
		Select("bb.id as benefit_id, bb.name as benefit_name, bb.benefit_type, "+
			"COUNT(*) as redemption_count, "+
			"DATE_TRUNC('month', redeemed_at)::date as month").
		Joins("JOIN brand_benefits bb ON bb.id = benefit_redemptions.benefit_id").
		Where("benefit_redemptions.brand_id = ?", brandID).
		Group("bb.id, bb.name, bb.benefit_type, DATE_TRUNC('month', redeemed_at)").
		Order("redemption_count DESC").
		Scan(&results).Error
	if err != nil {
		return nil, err
	}
	return results, nil
}

// GetCatalogStats returns item counts grouped by status and by type
func (r *DashboardRepository) GetCatalogStats(ctx context.Context, brandID uuid.UUID) (map[string]interface{}, error) {
	var byStatus []map[string]interface{}
	if err := r.getDB(ctx).Table("brand_items").
		Select("status as label, COUNT(*) as count").
		Where("brand_id = ?", brandID).
		Group("status").
		Order("count DESC").
		Scan(&byStatus).Error; err != nil {
		return nil, err
	}

	var byType []map[string]interface{}
	if err := r.getDB(ctx).Table("brand_items").
		Select("item_type as label, COUNT(*) as count").
		Where("brand_id = ?", brandID).
		Group("item_type").
		Order("count DESC").
		Scan(&byType).Error; err != nil {
		return nil, err
	}

	var totalCount int64
	if err := r.getDB(ctx).Table("brand_items").
		Where("brand_id = ?", brandID).
		Count(&totalCount).Error; err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"total_count": totalCount,
		"by_status":   byStatus,
		"by_type":     byType,
	}
	return result, nil
}

// GetCustomerSpendSegmentation returns customer count by spend range segments
func (r *DashboardRepository) GetCustomerSpendSegmentation(ctx context.Context, brandID uuid.UUID) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	err := r.getDB(ctx).Table("loyalty_accounts").
		Select("CASE "+
			"WHEN total_spend >= 0 AND total_spend < 1000000 THEN 'low' "+
			"WHEN total_spend >= 1000000 AND total_spend < 5000000 THEN 'medium' "+
			"WHEN total_spend >= 5000000 AND total_spend < 10000000 THEN 'high' "+
			"ELSE 'premium' END as segment, "+
			"COUNT(*) as customer_count").
		Where("brand_id = ?", brandID).
		Group("segment").
		Order("segment ASC").
		Scan(&results).Error
	if err != nil {
		return nil, err
	}
	return results, nil
}

// GetBrandNewCustomerCount returns count of customers created in a date range
func (r *DashboardRepository) GetBrandNewCustomerCount(ctx context.Context, brandID uuid.UUID, fromDate, toDate time.Time) (float64, error) {
	var count int64
	err := r.getDB(ctx).Table("brand_customers").
		Where("brand_id = ? AND created_at >= ? AND created_at <= ?", brandID, fromDate, toDate).
		Count(&count).Error
	if err != nil {
		return 0, err
	}
	return float64(count), nil
}

// GetBrandCustomerClaimRate returns the ratio of claimed customers
func (r *DashboardRepository) GetBrandCustomerClaimRate(ctx context.Context, brandID uuid.UUID) (float64, error) {
	var total, claimed int64
	if err := r.getDB(ctx).Table("brand_customers").Where("brand_id = ?", brandID).Count(&total).Error; err != nil {
		return 0, err
	}
	if err := r.getDB(ctx).Table("brand_customers").Where("brand_id = ? AND claimed_at IS NOT NULL", brandID).Count(&claimed).Error; err != nil {
		return 0, err
	}
	if total == 0 {
		return 0, nil
	}
	return float64(claimed) / float64(total) * 100, nil
}

// GetBrandRedemptionCount returns count of benefit redemptions in a date range
func (r *DashboardRepository) GetBrandRedemptionCount(ctx context.Context, brandID uuid.UUID, fromDate, toDate time.Time) (float64, error) {
	var count int64
	err := r.getDB(ctx).Table("benefit_redemptions").
		Where("brand_id = ? AND redeemed_at >= ? AND redeemed_at <= ?", brandID, fromDate, toDate).
		Count(&count).Error
	if err != nil {
		return 0, err
	}
	return float64(count), nil
}

// GetBrandItemCount returns total items count for a brand
func (r *DashboardRepository) GetBrandItemCount(ctx context.Context, brandID uuid.UUID) (float64, error) {
	var count int64
	err := r.getDB(ctx).Table("brand_items").
		Where("brand_id = ?", brandID).
		Count(&count).Error
	if err != nil {
		return 0, err
	}
	return float64(count), nil
}

// GetBrandItemCountByType returns item count grouped by type
func (r *DashboardRepository) GetBrandItemCountByType(ctx context.Context, brandID uuid.UUID) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	err := r.getDB(ctx).Table("brand_items").
		Select("item_type, COUNT(*) as count").
		Where("brand_id = ?", brandID).
		Group("item_type").
		Scan(&results).Error
	if err != nil {
		return nil, err
	}
	return results, nil
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
