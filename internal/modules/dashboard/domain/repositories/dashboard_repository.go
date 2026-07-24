package repositories

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/modules/dashboard/domain/entities"

	"github.com/google/uuid"
)

type IDashboardRepository interface {
	UpsertAdminSnapshot(ctx context.Context, snapshot *entities.DashboardAdminDailySnapshot) error
	GetAdminSnapshots(ctx context.Context, fromDate, toDate time.Time) ([]*entities.DashboardAdminDailySnapshot, error)
	GetLatestAdminSnapshot(ctx context.Context) (*entities.DashboardAdminDailySnapshot, error)

	UpsertBrandSnapshot(ctx context.Context, snapshot *entities.DashboardBrandDailySnapshot) error
	GetBrandSnapshots(ctx context.Context, brandID uuid.UUID, fromDate, toDate time.Time) ([]*entities.DashboardBrandDailySnapshot, error)
	GetLatestBrandSnapshot(ctx context.Context, brandID uuid.UUID) (*entities.DashboardBrandDailySnapshot, error)

	UpsertUserWardrobeSummary(ctx context.Context, summary *entities.DashboardUserWardrobeSummary) error
	GetUserWardrobeSummary(ctx context.Context, userID uuid.UUID) (*entities.DashboardUserWardrobeSummary, error)

	// Admin expansion APIs
	GetSubscriptionDistribution(ctx context.Context) ([]map[string]interface{}, error)
	GetRevenueBreakdown(ctx context.Context, fromDate, toDate time.Time) ([]map[string]interface{}, error)
	GetAIUsageByOperation(ctx context.Context, fromDate, toDate time.Time) ([]map[string]interface{}, error)
	GetAIErrorBreakdown(ctx context.Context) ([]map[string]interface{}, error)
	GetRenewalStats(ctx context.Context) (map[string]interface{}, error)
	GetTotalAIRequestCount(ctx context.Context, fromDate, toDate time.Time) (int64, error)

	// Brand expansion APIs
	GetCustomerAcquisition(ctx context.Context, brandID uuid.UUID) ([]map[string]interface{}, error)
	GetLoyaltyTierDistribution(ctx context.Context, brandID uuid.UUID) ([]map[string]interface{}, error)
	GetPointExpiryForecast(ctx context.Context, brandID uuid.UUID) ([]map[string]interface{}, error)
	GetBenefitRedemptionAnalytics(ctx context.Context, brandID uuid.UUID) ([]map[string]interface{}, error)
	GetCatalogStats(ctx context.Context, brandID uuid.UUID) (map[string]interface{}, error)
	GetCustomerSpendSegmentation(ctx context.Context, brandID uuid.UUID) ([]map[string]interface{}, error)

	// User expansion APIs
	GetWardrobeCategoryDistribution(ctx context.Context, userID uuid.UUID) ([]map[string]interface{}, error)

	// Brand Dynamic Query metrics (live query)
	GetBrandNewCustomerCount(ctx context.Context, brandID uuid.UUID, fromDate, toDate time.Time) (float64, error)
	GetBrandCustomerClaimRate(ctx context.Context, brandID uuid.UUID) (float64, error)
	GetBrandRedemptionCount(ctx context.Context, brandID uuid.UUID, fromDate, toDate time.Time) (float64, error)
	GetBrandItemCount(ctx context.Context, brandID uuid.UUID) (float64, error)
	GetBrandItemCountByType(ctx context.Context, brandID uuid.UUID) ([]map[string]interface{}, error)
}
