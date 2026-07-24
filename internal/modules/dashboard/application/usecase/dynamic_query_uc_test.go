package usecase

import (
	"context"
	"math"
	"testing"
	"time"

	"smart-wardrobe-be/internal/modules/brand/domain/constants/brandmemberrole"
	"smart-wardrobe-be/internal/modules/dashboard/application/dto"
	"smart-wardrobe-be/internal/modules/dashboard/domain/constants/granularity"
	"smart-wardrobe-be/internal/modules/dashboard/domain/constants/metrickey"
	"smart-wardrobe-be/internal/modules/dashboard/domain/constants/targetdashboard"
	"smart-wardrobe-be/internal/modules/dashboard/domain/entities"

	brandContract "smart-wardrobe-be/internal/modules/brand/presentation/contract"
	"smart-wardrobe-be/internal/shared/domain/constants/roleslug"

	"github.com/google/uuid"
)

type mockDynamicQueryBrandReader struct {
	brandContract.IBrandDashboardReader
	role brandmemberrole.BrandMemberRole
}

func (m *mockDynamicQueryBrandReader) GetUserBrandRole(ctx context.Context, userID uuid.UUID, brandID uuid.UUID) (brandmemberrole.BrandMemberRole, error) {
	return m.role, nil
}

type mockDynamicQueryRepo struct {
	adminSnapshots []*entities.DashboardAdminDailySnapshot
	brandSnapshots []*entities.DashboardBrandDailySnapshot
	userSummary    *entities.DashboardUserWardrobeSummary
}

func (m *mockDynamicQueryRepo) UpsertAdminSnapshot(ctx context.Context, snapshot *entities.DashboardAdminDailySnapshot) error {
	return nil
}

func (m *mockDynamicQueryRepo) GetAdminSnapshots(ctx context.Context, fromDate, toDate time.Time) ([]*entities.DashboardAdminDailySnapshot, error) {
	return m.adminSnapshots, nil
}

func (m *mockDynamicQueryRepo) GetLatestAdminSnapshot(ctx context.Context) (*entities.DashboardAdminDailySnapshot, error) {
	return nil, nil
}

func (m *mockDynamicQueryRepo) UpsertBrandSnapshot(ctx context.Context, snapshot *entities.DashboardBrandDailySnapshot) error {
	return nil
}

func (m *mockDynamicQueryRepo) GetBrandSnapshots(ctx context.Context, brandID uuid.UUID, fromDate, toDate time.Time) ([]*entities.DashboardBrandDailySnapshot, error) {
	return m.brandSnapshots, nil
}

func (m *mockDynamicQueryRepo) GetLatestBrandSnapshot(ctx context.Context, brandID uuid.UUID) (*entities.DashboardBrandDailySnapshot, error) {
	return nil, nil
}

func (m *mockDynamicQueryRepo) UpsertUserWardrobeSummary(ctx context.Context, summary *entities.DashboardUserWardrobeSummary) error {
	return nil
}

func (m *mockDynamicQueryRepo) GetUserWardrobeSummary(ctx context.Context, userID uuid.UUID) (*entities.DashboardUserWardrobeSummary, error) {
	return m.userSummary, nil
}

// Expansion stub methods
func (m *mockDynamicQueryRepo) GetSubscriptionDistribution(ctx context.Context) ([]map[string]interface{}, error) {
	return nil, nil
}
func (m *mockDynamicQueryRepo) GetRevenueBreakdown(ctx context.Context, fromDate, toDate time.Time) ([]map[string]interface{}, error) {
	return nil, nil
}
func (m *mockDynamicQueryRepo) GetAIUsageByOperation(ctx context.Context, fromDate, toDate time.Time) ([]map[string]interface{}, error) {
	return nil, nil
}
func (m *mockDynamicQueryRepo) GetAIErrorBreakdown(ctx context.Context) ([]map[string]interface{}, error) {
	return nil, nil
}
func (m *mockDynamicQueryRepo) GetRenewalStats(ctx context.Context) (map[string]interface{}, error) {
	return nil, nil
}
func (m *mockDynamicQueryRepo) GetCustomerAcquisition(ctx context.Context, brandID uuid.UUID) ([]map[string]interface{}, error) {
	return nil, nil
}
func (m *mockDynamicQueryRepo) GetLoyaltyTierDistribution(ctx context.Context, brandID uuid.UUID) ([]map[string]interface{}, error) {
	return nil, nil
}
func (m *mockDynamicQueryRepo) GetPointExpiryForecast(ctx context.Context, brandID uuid.UUID) ([]map[string]interface{}, error) {
	return nil, nil
}
func (m *mockDynamicQueryRepo) GetBenefitRedemptionAnalytics(ctx context.Context, brandID uuid.UUID) ([]map[string]interface{}, error) {
	return nil, nil
}
func (m *mockDynamicQueryRepo) GetCatalogStats(ctx context.Context, brandID uuid.UUID) (map[string]interface{}, error) {
	return nil, nil
}
func (m *mockDynamicQueryRepo) GetCustomerSpendSegmentation(ctx context.Context, brandID uuid.UUID) ([]map[string]interface{}, error) {
	return nil, nil
}
func (m *mockDynamicQueryRepo) GetWardrobeCategoryDistribution(ctx context.Context, userID uuid.UUID) ([]map[string]interface{}, error) {
	return nil, nil
}
func (m *mockDynamicQueryRepo) GetBrandNewCustomerCount(ctx context.Context, brandID uuid.UUID, fromDate, toDate time.Time) (float64, error) {
	return 0, nil
}
func (m *mockDynamicQueryRepo) GetBrandCustomerClaimRate(ctx context.Context, brandID uuid.UUID) (float64, error) {
	return 0, nil
}
func (m *mockDynamicQueryRepo) GetBrandRedemptionCount(ctx context.Context, brandID uuid.UUID, fromDate, toDate time.Time) (float64, error) {
	return 0, nil
}
func (m *mockDynamicQueryRepo) GetBrandItemCount(ctx context.Context, brandID uuid.UUID) (float64, error) {
	return 0, nil
}
func (m *mockDynamicQueryRepo) GetBrandItemCountByType(ctx context.Context, brandID uuid.UUID) ([]map[string]interface{}, error) {
	return nil, nil
}
func (m *mockDynamicQueryRepo) GetTotalAIRequestCount(ctx context.Context, fromDate, toDate time.Time) (int64, error) {
	return 0, nil
}

func compareFloat64(t *testing.T, name string, got, expected float64) {
	t.Helper()
	if math.Abs(got-expected) > 0.01 {
		t.Errorf("%s: expected %.2f, got %.2f", name, expected, got)
	}
}

func TestDynamicQuery_AdminTarget(t *testing.T) {
	userID := uuid.New()
	now := time.Now().Truncate(24 * time.Hour)

	repo := &mockDynamicQueryRepo{
		adminSnapshots: []*entities.DashboardAdminDailySnapshot{
			{SnapshotDate: now.AddDate(0, 0, -1), TotalUsers: 100, ActiveUsersDau: 60, AIRequestsCount: 500, SubscriptionRevenueVND: 1000000, NetAIMarginVND: 600000},
			{SnapshotDate: now, TotalUsers: 110, ActiveUsersDau: 65, AIRequestsCount: 550, SubscriptionRevenueVND: 1100000, NetAIMarginVND: 650000},
		},
	}

	uc := NewDynamicQueryUseCase(repo, &mockDynamicQueryBrandReader{role: ""})
	res, err := uc.ExecuteDynamicQuery(context.Background(), userID, roleslug.Admin, &dto.DynamicQueryReq{
		TargetDashboard: targetdashboard.Admin,
		Metrics:         []metrickey.MetricKey{metrickey.TotalUsers, metrickey.AIRequestsCount},
		Timeframe:       dto.DynamicQueryTimeframeDTO{FromDate: now.AddDate(0, 0, -2).Format("2006-01-02"), ToDate: now.Format("2006-01-02")},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.TargetDashboard != targetdashboard.Admin {
		t.Errorf("expected Admin target, got %s", res.TargetDashboard)
	}
	if len(res.Series) != 2 {
		t.Fatalf("expected 2 series points, got %d", len(res.Series))
	}
	if res.Series[0].Metrics[metrickey.TotalUsers] != 100 {
		t.Errorf("expected first point TotalUsers=100, got %v", res.Series[0].Metrics[metrickey.TotalUsers])
	}
}

func TestDynamicQuery_AdminRBAC_ForbiddenForUser(t *testing.T) {
	userID := uuid.New()
	uc := NewDynamicQueryUseCase(&mockDynamicQueryRepo{}, &mockDynamicQueryBrandReader{role: ""})
	_, err := uc.ExecuteDynamicQuery(context.Background(), userID, roleslug.User, &dto.DynamicQueryReq{
		TargetDashboard: targetdashboard.Admin,
		Metrics:         []metrickey.MetricKey{metrickey.TotalUsers},
		Timeframe:       dto.DynamicQueryTimeframeDTO{FromDate: "2026-07-01", ToDate: "2026-07-23"},
	})
	if err == nil {
		t.Fatal("expected Forbidden error for non-admin accessing admin dashboard")
	}
}

func TestDynamicQuery_CompareWithPrevious(t *testing.T) {
	adminID := uuid.New()
	now := time.Now().Truncate(24 * time.Hour)

	repo := &mockDynamicQueryRepo{
		adminSnapshots: []*entities.DashboardAdminDailySnapshot{
			{SnapshotDate: now.AddDate(0, 0, -1), TotalUsers: 200, AIRequestsCount: 1000},
			{SnapshotDate: now, TotalUsers: 220, AIRequestsCount: 1100},
		},
	}

	uc := NewDynamicQueryUseCase(repo, &mockDynamicQueryBrandReader{})
	res, err := uc.ExecuteDynamicQuery(context.Background(), adminID, roleslug.Admin, &dto.DynamicQueryReq{
		TargetDashboard:     targetdashboard.Admin,
		Metrics:             []metrickey.MetricKey{metrickey.TotalUsers, metrickey.AIRequestsCount},
		Timeframe:           dto.DynamicQueryTimeframeDTO{FromDate: now.AddDate(0, 0, -1).Format("2006-01-02"), ToDate: now.Format("2006-01-02")},
		CompareWithPrevious: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Comparison == nil {
		t.Fatal("expected Comparison to be non-nil when compareWithPrevious=true")
	}
	if res.Comparison.PreviousPeriodFrom == "" || res.Comparison.PreviousPeriodTo == "" {
		t.Error("expected previous period dates to be set")
	}
}

func TestDynamicQuery_LargeDateRange_AutoDowngrade(t *testing.T) {
	tests := []struct {
		name      string
		days      int
		inputGran granularity.Granularity
		wantGran  granularity.Granularity
	}{
		{"daily >730d → month", 800, granularity.Day, granularity.Month},
		{"daily 100d → keep day", 100, granularity.Day, granularity.Day},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fromDate := time.Now().AddDate(0, 0, -tt.days)
			toDate := time.Now()
			got := enforceMaxGranularity(fromDate, toDate, tt.inputGran)
			if got != tt.wantGran {
				t.Errorf("enforceMaxGranularity(%d, %s) = %s, want %s", tt.days, tt.inputGran, got, tt.wantGran)
			}
		})
	}
}
