package entities

import (
	"time"

	sharedEntities "smart-wardrobe-be/internal/shared/domain/entities"

	"github.com/google/uuid"
)

type DashboardAdminDailySnapshot struct {
	sharedEntities.AuditableEntity
	SnapshotDate           time.Time `gorm:"column:snapshot_date;type:date;not null;uniqueIndex"`
	TotalUsers             int       `gorm:"column:total_users;not null;default:0"`
	NewUsers               int       `gorm:"column:new_users;not null;default:0"`
	ActiveUsersDau         int       `gorm:"column:active_users_dau;not null;default:0"`
	TotalBrands            int       `gorm:"column:total_brands;not null;default:0"`
	PendingReviewBrands    int       `gorm:"column:pending_review_brands;not null;default:0"`
	AIRequestsCount        int64     `gorm:"column:ai_requests_count;not null;default:0"`
	AIErrorCount           int64     `gorm:"column:ai_error_count;not null;default:0"`
	AIPaidTokens           int64     `gorm:"column:ai_paid_tokens;not null;default:0"`
	AIActualCostMicroVND   int64     `gorm:"column:ai_actual_cost_micro_vnd;not null;default:0"`
	SubscriptionRevenueVND float64   `gorm:"column:subscription_revenue_vnd;type:numeric(12,2);not null;default:0"`
	NetAIMarginVND         float64   `gorm:"column:net_ai_margin_vnd;type:numeric(12,2);not null;default:0"`
	PayOSTransactionCount  int       `gorm:"column:payos_transaction_count;not null;default:0"`
	PayOSSuccessCount      int       `gorm:"column:payos_success_count;not null;default:0"`
}

func (DashboardAdminDailySnapshot) TableName() string {
	return "dashboard_admin_daily_snapshots"
}

type DashboardBrandDailySnapshot struct {
	sharedEntities.AuditableEntity
	BrandID                   uuid.UUID `gorm:"column:brand_id;type:uuid;not null;index"`
	SnapshotDate              time.Time `gorm:"column:snapshot_date;type:date;not null"`
	TotalMembers              int       `gorm:"column:total_members;not null;default:0"`
	PointsIssued              int       `gorm:"column:points_issued;not null;default:0"`
	PointsRedeemed            int       `gorm:"column:points_redeemed;not null;default:0"`
	PointsExpiring30d         int       `gorm:"column:points_expiring_30d;not null;default:0"`
	PointsLiabilityValueVND   float64   `gorm:"column:points_liability_value_vnd;type:numeric(12,2);not null;default:0"`
	ActiveVouchers            int       `gorm:"column:active_vouchers;not null;default:0"`
	VouchersRedeemed          int       `gorm:"column:vouchers_redeemed;not null;default:0"`
	DigitalSampleVotesCount   int       `gorm:"column:digital_sample_votes_count;not null;default:0"`
	DigitalSampleAvgRating    float64   `gorm:"column:digital_sample_avg_rating;type:numeric(3,2);not null;default:0"`
	OpenCSTickets             int       `gorm:"column:open_cs_tickets;not null;default:0"`
	CSFirstResponseAvgSeconds int       `gorm:"column:cs_first_response_avg_seconds;not null;default:0"`
}

func (DashboardBrandDailySnapshot) TableName() string {
	return "dashboard_brand_daily_snapshots"
}

type DashboardUserWardrobeSummary struct {
	UserID                  uuid.UUID                   `gorm:"column:user_id;type:uuid;primaryKey"`
	CreatedAt               time.Time                   `gorm:"column:created_at;type:timestamp with time zone;not null;default:now()"`
	TotalItems              int                         `gorm:"column:total_items;not null;default:0"`
	TotalWardrobeValueVND   float64                     `gorm:"column:total_wardrobe_value_vnd;type:numeric(12,2);not null;default:0"`
	AverageCostPerWearVND   float64                     `gorm:"column:average_cost_per_wear_vnd;type:numeric(12,2);not null;default:0"`
	UnderutilizedItemsCount int                         `gorm:"column:underutilized_items_count;not null;default:0"`
	UnderutilizedItems      sharedEntities.JSONDocument `gorm:"column:underutilized_items;type:jsonb;not null;default:'[]'"`
	AIQuotaRemaining        int                         `gorm:"column:ai_quota_remaining;not null;default:0"`
	ExpiringVouchersCount   int                         `gorm:"column:expiring_vouchers_count;not null;default:0"`
	UpdatedAt               time.Time                   `gorm:"column:updated_at;type:timestamp with time zone;not null;default:now()"`
}

func (DashboardUserWardrobeSummary) TableName() string {
	return "dashboard_user_wardrobe_summaries"
}
