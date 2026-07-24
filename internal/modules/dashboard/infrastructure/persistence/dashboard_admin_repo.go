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
			"pending_review_brands", "ai_requests_count",
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

// GetSubscriptionDistribution returns user count grouped by plan code
func (r *DashboardRepository) GetSubscriptionDistribution(ctx context.Context) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	err := r.getDB(ctx).Table("user_subscriptions").
		Select("current_plan_code as plan_code, COUNT(*) as user_count").
		Group("current_plan_code").
		Order("user_count DESC").
		Scan(&results).Error
	if err != nil {
		return nil, err
	}
	return results, nil
}

// GetRevenueBreakdown returns revenue grouped by plan code and month
func (r *DashboardRepository) GetRevenueBreakdown(ctx context.Context, fromDate, toDate time.Time) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	query := r.getDB(ctx).Table("deposit_transactions").
		Select("plan_code, DATE_TRUNC('month', created_at)::date as month, COUNT(*) as transaction_count, SUM(amount) as revenue").
		Where("status = 1 AND transaction_type = 'DIRECT_PURCHASE'")

	if !fromDate.IsZero() {
		query = query.Where("created_at >= ?", fromDate)
	}
	if !toDate.IsZero() {
		query = query.Where("created_at <= ?", toDate)
	}

	err := query.Group("plan_code, DATE_TRUNC('month', created_at)").
		Order("month ASC, plan_code ASC").
		Scan(&results).Error
	if err != nil {
		return nil, err
	}
	return results, nil
}

// GetAIUsageByOperation returns AI usage grouped by operation and logical route
func (r *DashboardRepository) GetAIUsageByOperation(ctx context.Context, fromDate, toDate time.Time) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	query := r.getDB(ctx).Table("ai_usage_events").
		Select("operation, logical_route, COUNT(*) as request_count, " +
			"COALESCE(SUM(actual_cost_micro_vnd), 0) as total_cost_micro_vnd, " +
			"COALESCE(SUM(prompt_tokens), 0) as total_prompt_tokens, " +
			"COALESCE(SUM(output_tokens), 0) as total_output_tokens")

	if !fromDate.IsZero() {
		query = query.Where("created_at >= ?", fromDate)
	}
	if !toDate.IsZero() {
		query = query.Where("created_at <= ?", toDate)
	}

	err := query.Group("operation, logical_route").
		Order("request_count DESC").
		Scan(&results).Error
	if err != nil {
		return nil, err
	}
	return results, nil
}

// GetTotalAIRequestCount returns total AI requests within a date range
func (r *DashboardRepository) GetTotalAIRequestCount(ctx context.Context, fromDate, toDate time.Time) (int64, error) {
	var count int64
	query := r.getDB(ctx).Table("ai_usage_events").Model(nil)
	if !fromDate.IsZero() {
		query = query.Where("created_at >= ?", fromDate)
	}
	if !toDate.IsZero() {
		query = query.Where("created_at <= ?", toDate)
	}
	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// GetAIErrorBreakdown returns error count grouped by error_code, provider, model
func (r *DashboardRepository) GetAIErrorBreakdown(ctx context.Context) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	err := r.getDB(ctx).Table("ai_usage_events").
		Select("error_code, provider, model, COUNT(*) as error_count").
		Where("status IN ?", []string{"FAILED", "RELEASED", "EXPIRED"}).
		Group("error_code, provider, model").
		Order("error_count DESC").
		Scan(&results).Error
	if err != nil {
		return nil, err
	}
	return results, nil
}

// GetRenewalStats returns renewal attempt statistics
func (r *DashboardRepository) GetRenewalStats(ctx context.Context) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := r.getDB(ctx).Table("subscription_renewal_attempts").
		Select("COUNT(*) as total_attempts, " +
			"COALESCE(SUM(CASE WHEN status = 'SUCCESS' THEN 1 ELSE 0 END), 0) as success_count, " +
			"COALESCE(SUM(CASE WHEN status = 'FAILED' THEN 1 ELSE 0 END), 0) as failed_count, " +
			"COALESCE(AVG(attempt_count), 0) as avg_retry_count").
		Scan(&result).Error
	if err != nil {
		return nil, err
	}
	return result, nil
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
