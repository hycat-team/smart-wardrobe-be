package entities

import (
	"time"

	sharedEntities "smart-wardrobe-be/internal/shared/domain/entities"

	"github.com/google/uuid"
)

type DashboardAdminDailySnapshot struct {
	sharedEntities.AuditableEntity
	SnapshotDate           time.Time `gorm:"type:date;not null;uniqueIndex"`
	TotalUsers             int       `gorm:"not null;default:0"`
	NewUsers               int       `gorm:"not null;default:0"`
	ActiveUsersDau         int       `gorm:"not null;default:0"`
	TotalBrands            int       `gorm:"not null;default:0"`
	PendingReviewBrands    int       `gorm:"not null;default:0"`
	ActiveCampaigns        int       `gorm:"not null;default:0"`
	AIRequestsCount        int       `gorm:"not null;default:0"`
	AIErrorCount           int       `gorm:"not null;default:0"`
	AIPaidTokens           int64     `gorm:"not null;default:0"`
	AIActualCostMicroVND   int64     `gorm:"not null;default:0"`
	SubscriptionRevenueVND float64   `gorm:"type:numeric(12,2);not null;default:0"`
	NetAIMarginVND         float64   `gorm:"type:numeric(12,2);not null;default:0"`
	PayOSTransactionCount  int       `gorm:"not null;default:0"`
	PayOSSuccessCount      int       `gorm:"not null;default:0"`
}

func (DashboardAdminDailySnapshot) TableName() string {
	return "dashboard_admin_daily_snapshots"
}

type DashboardBrandDailySnapshot struct {
	sharedEntities.AuditableEntity
	BrandID                   uuid.UUID `gorm:"type:uuid;not null;index"`
	SnapshotDate              time.Time `gorm:"type:date;not null"`
	TotalMembers              int       `gorm:"not null;default:0"`
	PointsIssued              int       `gorm:"not null;default:0"`
	PointsRedeemed            int       `gorm:"not null;default:0"`
	PointsExpiring30d         int       `gorm:"not null;default:0"`
	PointsLiabilityValueVND   float64   `gorm:"type:numeric(12,2);not null;default:0"`
	ActiveVouchers            int       `gorm:"not null;default:0"`
	VouchersRedeemed          int       `gorm:"not null;default:0"`
	DigitalSampleVotesCount   int       `gorm:"not null;default:0"`
	DigitalSampleAvgRating    float64   `gorm:"type:numeric(3,2);not null;default:0"`
	OpenCSTickets             int       `gorm:"not null;default:0"`
	CSFirstResponseAvgSeconds int       `gorm:"not null;default:0"`
}

func (DashboardBrandDailySnapshot) TableName() string {
	return "dashboard_brand_daily_snapshots"
}

type DashboardUserWardrobeSummary struct {
	UserID                  uuid.UUID                   `gorm:"type:uuid;primaryKey"`
	TotalItems              int                         `gorm:"not null;default:0"`
	TotalWardrobeValueVND   float64                     `gorm:"type:numeric(12,2);not null;default:0"`
	AverageCostPerWearVND   float64                     `gorm:"type:numeric(12,2);not null;default:0"`
	UnderutilizedItemsCount int                         `gorm:"not null;default:0"`
	CategoryDistribution    sharedEntities.JSONDocument `gorm:"type:jsonb;not null;default:'{}'"`
	ColorDistribution       sharedEntities.JSONDocument `gorm:"type:jsonb;not null;default:'{}'"`
	AIQuotaRemaining        int                         `gorm:"not null;default:0"`
	ExpiringVouchersCount   int                         `gorm:"not null;default:0"`
	UpdatedAt               time.Time                   `gorm:"type:timestamp with time zone;not null;default:now()"`
}

func (DashboardUserWardrobeSummary) TableName() string {
	return "dashboard_user_wardrobe_summaries"
}
