package usecase

import (
	"context"
	"testing"
	"time"

	"smart-wardrobe-be/services/commerce/internal/modules/dashboard/domain/entities"

	aiContract "smart-wardrobe-be/services/ai/contract/aiexecutor"
	aiDTO "smart-wardrobe-be/services/ai/dto/aiexecutor"

	brandDTO "smart-wardrobe-be/services/commerce/internal/modules/brand/application/dto"
	brandContract "smart-wardrobe-be/services/commerce/internal/modules/brand/presentation/contract"

	identityContract "smart-wardrobe-be/services/account/contract/identity"
	identityDTO "smart-wardrobe-be/services/account/dto/identity"

	subscriptionContract "smart-wardrobe-be/services/account/contract/subscription"
	subscriptionDTO "smart-wardrobe-be/services/account/dto/subscription"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type mockAdminDashboardRepo struct {
	latestSnapshot      *entities.DashboardAdminDailySnapshot
	snapshots           []*entities.DashboardAdminDailySnapshot
	upsertAdminCalled   bool
	latestSnapshotError error
	snapshotsError      error
}

func (m *mockAdminDashboardRepo) UpsertAdminSnapshot(ctx context.Context, snapshot *entities.DashboardAdminDailySnapshot) error {
	m.upsertAdminCalled = true
	return nil
}

func (m *mockAdminDashboardRepo) GetAdminSnapshots(ctx context.Context, fromDate, toDate time.Time) ([]*entities.DashboardAdminDailySnapshot, error) {
	if m.snapshotsError != nil {
		return nil, m.snapshotsError
	}
	return m.snapshots, nil
}

func (m *mockAdminDashboardRepo) GetLatestAdminSnapshot(ctx context.Context) (*entities.DashboardAdminDailySnapshot, error) {
	if m.latestSnapshotError != nil {
		return nil, m.latestSnapshotError
	}
	return m.latestSnapshot, nil
}

func (m *mockAdminDashboardRepo) UpsertBrandSnapshot(ctx context.Context, snapshot *entities.DashboardBrandDailySnapshot) error {
	return nil
}

func (m *mockAdminDashboardRepo) GetBrandSnapshots(ctx context.Context, brandID uuid.UUID, fromDate, toDate time.Time) ([]*entities.DashboardBrandDailySnapshot, error) {
	return nil, nil
}

func (m *mockAdminDashboardRepo) GetLatestBrandSnapshot(ctx context.Context, brandID uuid.UUID) (*entities.DashboardBrandDailySnapshot, error) {
	return nil, nil
}

func (m *mockAdminDashboardRepo) UpsertUserWardrobeSummary(ctx context.Context, summary *entities.DashboardUserWardrobeSummary) error {
	return nil
}

func (m *mockAdminDashboardRepo) GetUserWardrobeSummary(ctx context.Context, userID uuid.UUID) (*entities.DashboardUserWardrobeSummary, error) {
	return nil, nil
}

// Expansion stub methods
func (m *mockAdminDashboardRepo) GetSubscriptionDistribution(ctx context.Context) ([]map[string]interface{}, error) {
	return nil, nil
}
func (m *mockAdminDashboardRepo) GetRevenueBreakdown(ctx context.Context, fromDate, toDate time.Time) ([]map[string]interface{}, error) {
	return nil, nil
}
func (m *mockAdminDashboardRepo) GetAIUsageByOperation(ctx context.Context, fromDate, toDate time.Time) ([]map[string]interface{}, error) {
	return nil, nil
}
func (m *mockAdminDashboardRepo) GetAIErrorBreakdown(ctx context.Context) ([]map[string]interface{}, error) {
	return nil, nil
}
func (m *mockAdminDashboardRepo) GetRenewalStats(ctx context.Context) (map[string]interface{}, error) {
	return nil, nil
}
func (m *mockAdminDashboardRepo) GetCustomerAcquisition(ctx context.Context, brandID uuid.UUID) ([]map[string]interface{}, error) {
	return nil, nil
}
func (m *mockAdminDashboardRepo) GetLoyaltyTierDistribution(ctx context.Context, brandID uuid.UUID) ([]map[string]interface{}, error) {
	return nil, nil
}
func (m *mockAdminDashboardRepo) GetPointExpiryForecast(ctx context.Context, brandID uuid.UUID) ([]map[string]interface{}, error) {
	return nil, nil
}
func (m *mockAdminDashboardRepo) GetBenefitRedemptionAnalytics(ctx context.Context, brandID uuid.UUID) ([]map[string]interface{}, error) {
	return nil, nil
}
func (m *mockAdminDashboardRepo) GetCatalogStats(ctx context.Context, brandID uuid.UUID) (map[string]interface{}, error) {
	return nil, nil
}
func (m *mockAdminDashboardRepo) GetCustomerSpendSegmentation(ctx context.Context, brandID uuid.UUID) ([]map[string]interface{}, error) {
	return nil, nil
}
func (m *mockAdminDashboardRepo) GetWardrobeCategoryDistribution(ctx context.Context, userID uuid.UUID) ([]map[string]interface{}, error) {
	return nil, nil
}
func (m *mockAdminDashboardRepo) GetBrandNewCustomerCount(ctx context.Context, brandID uuid.UUID, fromDate, toDate time.Time) (float64, error) {
	return 0, nil
}
func (m *mockAdminDashboardRepo) GetBrandCustomerClaimRate(ctx context.Context, brandID uuid.UUID) (float64, error) {
	return 0, nil
}
func (m *mockAdminDashboardRepo) GetBrandRedemptionCount(ctx context.Context, brandID uuid.UUID, fromDate, toDate time.Time) (float64, error) {
	return 0, nil
}
func (m *mockAdminDashboardRepo) GetBrandItemCount(ctx context.Context, brandID uuid.UUID) (float64, error) {
	return 0, nil
}
func (m *mockAdminDashboardRepo) GetBrandItemCountByType(ctx context.Context, brandID uuid.UUID) ([]map[string]interface{}, error) {
	return nil, nil
}
func (m *mockAdminDashboardRepo) GetTotalAIRequestCount(ctx context.Context, fromDate, toDate time.Time) (int64, error) {
	return 0, nil
}

type mockUserReader struct {
	identityContract.IUserDashboardReader
	stats *identityDTO.UserAdminStatsDTO
	err   error
}

func (m *mockUserReader) GetAdminUserStats(ctx context.Context, snapshotDate time.Time) (*identityDTO.UserAdminStatsDTO, error) {
	return m.stats, m.err
}

type mockBrandStatsReader struct {
	brandContract.IBrandDashboardReader
	stats *brandDTO.BrandAdminStatsDTO
	err   error
}

func (m *mockBrandStatsReader) GetAdminBrandStats(ctx context.Context, snapshotDate time.Time) (*brandDTO.BrandAdminStatsDTO, error) {
	return m.stats, m.err
}

type mockAICostReader struct {
	aiContract.IAICostDashboardReader
	stats *aiDTO.AIUsageAdminStatsDTO
	err   error
}

func (m *mockAICostReader) GetAdminAIUsageStats(ctx context.Context, snapshotDate time.Time) (*aiDTO.AIUsageAdminStatsDTO, error) {
	return m.stats, m.err
}

func (m *mockAICostReader) GetAIEventsDrillDown(ctx context.Context, status string, page, limit int) ([]aiDTO.AIEventDrillDownDTO, int64, error) {
	return nil, 0, nil
}

type mockSubscriptionReader struct {
	subscriptionContract.ISubscriptionDashboardReader
	stats   *subscriptionDTO.AdminSubscriptionStatsDTO
	renewal *subscriptionDTO.RenewalStatsDTO
	err     error
}

func (m *mockSubscriptionReader) GetAdminSubscriptionStats(ctx context.Context, snapshotDate time.Time) (*subscriptionDTO.AdminSubscriptionStatsDTO, error) {
	return m.stats, m.err
}

func (m *mockSubscriptionReader) GetRenewalStats(ctx context.Context) (*subscriptionDTO.RenewalStatsDTO, error) {
	return m.renewal, m.err
}

func TestAdminGetKPICards_FromSnapshot(t *testing.T) {
	repo := &mockAdminDashboardRepo{
		latestSnapshot: &entities.DashboardAdminDailySnapshot{
			SnapshotDate:           time.Now().Truncate(24 * time.Hour),
			TotalUsers:             1000,
			ActiveUsersDau:         500,
			TotalBrands:            50,
			PendingReviewBrands:    3,
			SubscriptionRevenueVND: 1000000,
			NetAIMarginVND:         500000,
			PayOSSuccessCount:      95,
		},
	}

	uc := NewAdminDashboardUseCase(repo, &mockUserReader{}, &mockBrandStatsReader{}, &mockAICostReader{}, &mockSubscriptionReader{})
	res, err := uc.GetAdminKPICards(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.TotalUsers != 1000 {
		t.Errorf("expected TotalUsers=1000, got %d", res.TotalUsers)
	}
	if res.ActiveUsersDAU != 500 {
		t.Errorf("expected ActiveUsersDAU=500, got %d", res.ActiveUsersDAU)
	}
	if res.NetAIMarginVND != 500000 {
		t.Errorf("expected NetAIMarginVND=500000, got %v", res.NetAIMarginVND)
	}
}

func TestAdminGetKPICards_FallbackToLiveReader(t *testing.T) {
	repo := &mockAdminDashboardRepo{
		latestSnapshot: nil,
	}
	userReader := &mockUserReader{
		stats: &identityDTO.UserAdminStatsDTO{
			TotalUsers:     800,
			ActiveUsersDau: 400,
		},
	}
	brandReader := &mockBrandStatsReader{
		stats: &brandDTO.BrandAdminStatsDTO{
			TotalBrands:         30,
			PendingReviewBrands: 2,
		},
	}
	aiReader := &mockAICostReader{
		stats: &aiDTO.AIUsageAdminStatsDTO{
			AIRequestsCount:      1000,
			AIActualCostMicroVND: 500000,
			AIPaidTokens:         10000,
			AIErrorCount:         5,
		},
	}
	subscriptionReader := &mockSubscriptionReader{
		stats: &subscriptionDTO.AdminSubscriptionStatsDTO{
			TotalSubscriptionRevenue: decimal.NewFromFloat(20000000),
			PayOSTransactionCount:    120,
			PayOSSuccessCount:        115,
		},
	}

	uc := NewAdminDashboardUseCase(repo, userReader, brandReader, aiReader, subscriptionReader)
	res, err := uc.GetAdminKPICards(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.TotalUsers != 800 {
		t.Errorf("expected TotalUsers=800, got %d", res.TotalUsers)
	}
	if res.TotalBrands != 30 {
		t.Errorf("expected TotalBrands=30, got %d", res.TotalBrands)
	}
	expectedNetMargin := -0.5
	if res.NetAIMarginVND != expectedNetMargin {
		t.Errorf("expected NetAIMarginVND=%v, got %v", expectedNetMargin, res.NetAIMarginVND)
	}
}

func TestAdminGetKPICards_NoData(t *testing.T) {
	repo := &mockAdminDashboardRepo{
		latestSnapshot: nil,
	}
	uc := NewAdminDashboardUseCase(repo, &mockUserReader{}, &mockBrandStatsReader{}, &mockAICostReader{}, &mockSubscriptionReader{})
	res, err := uc.GetAdminKPICards(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.TotalUsers != 0 || res.ActiveUsersDAU != 0 || res.TotalBrands != 0 {
		t.Error("expected all KPI fields to be 0 when no data")
	}
}

func TestAdminGetAIMarginAnalytics_FromSnapshots(t *testing.T) {
	now := time.Now()
	snapshots := []*entities.DashboardAdminDailySnapshot{
		{
			SnapshotDate:           now.AddDate(0, 0, -2),
			SubscriptionRevenueVND: 1000000,
			NetAIMarginVND:         600000,
			AIPaidTokens:           5000,
			AIRequestsCount:        100,
			AIErrorCount:           2,
		},
		{
			SnapshotDate:           now.AddDate(0, 0, -1),
			SubscriptionRevenueVND: 2000000,
			NetAIMarginVND:         1200000,
			AIPaidTokens:           8000,
			AIRequestsCount:        150,
			AIErrorCount:           3,
		},
	}

	repo := &mockAdminDashboardRepo{
		snapshots: snapshots,
	}
	uc := NewAdminDashboardUseCase(repo, &mockUserReader{}, &mockBrandStatsReader{}, &mockAICostReader{}, &mockSubscriptionReader{})
	res, err := uc.GetAIMarginAnalytics(context.Background(), now.AddDate(0, 0, -3), now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.TotalSubscriptionRevVND != 3000000 {
		t.Errorf("expected TotalSubscriptionRevVND=3000000, got %v", res.TotalSubscriptionRevVND)
	}
	if res.TotalAIActualCostVND != 1200000 {
		t.Errorf("expected TotalAIActualCostVND=1200000, got %v", res.TotalAIActualCostVND)
	}
	if res.MarginPercentage != 60.0 {
		t.Errorf("expected MarginPercentage=60, got %v", res.MarginPercentage)
	}
}

func TestAdminGetPayOSReconciliation(t *testing.T) {
	repo := &mockAdminDashboardRepo{
		latestSnapshot: &entities.DashboardAdminDailySnapshot{
			PayOSTransactionCount:  100,
			PayOSSuccessCount:      95,
			PayOSPendingCount:      3,
			PayOSFailedCount:       1,
			PayOSVoidedCount:       1,
			SubscriptionRevenueVND: 5000000,
		},
	}
	uc := NewAdminDashboardUseCase(repo, &mockUserReader{}, &mockBrandStatsReader{}, &mockAICostReader{}, &mockSubscriptionReader{})
	res, err := uc.GetPayOSReconciliation(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.TotalTransactions != 100 {
		t.Errorf("expected TotalTransactions=100, got %d", res.TotalTransactions)
	}
	if res.SuccessCount != 95 {
		t.Errorf("expected SuccessCount=95, got %d", res.SuccessCount)
	}
	if res.FailedCount != 1 {
		t.Errorf("expected FailedCount=1 (real failures, not total − success), got %d", res.FailedCount)
	}
	if res.PendingCount != 3 || res.VoidedCount != 1 {
		t.Errorf("expected PendingCount=3 and VoidedCount=1, got %+v", res)
	}
}

func TestAdminGetSystemOverview(t *testing.T) {
	now := time.Now()
	repo := &mockAdminDashboardRepo{
		latestSnapshot: &entities.DashboardAdminDailySnapshot{
			SnapshotDate:           now.Truncate(24 * time.Hour),
			TotalUsers:             1000,
			ActiveUsersDau:         500,
			SubscriptionRevenueVND: 2000000,
			NetAIMarginVND:         1000000,
			PayOSTransactionCount:  100,
			PayOSSuccessCount:      95,
		},
		snapshots: []*entities.DashboardAdminDailySnapshot{
			{
				SnapshotDate:           now.AddDate(0, 0, -1),
				SubscriptionRevenueVND: 2000000,
				NetAIMarginVND:         1000000,
				AIPaidTokens:           5000,
				AIRequestsCount:        100,
				AIErrorCount:           2,
			},
		},
	}
	uc := NewAdminDashboardUseCase(repo, &mockUserReader{}, &mockBrandStatsReader{}, &mockAICostReader{}, &mockSubscriptionReader{})
	res, err := uc.GetSystemOverview(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.KPICards == nil {
		t.Fatal("expected KPICards to be non-nil")
	}
	if res.KPICards.TotalUsers != 1000 {
		t.Errorf("expected TotalUsers=1000, got %d", res.KPICards.TotalUsers)
	}
	if res.PayOSReconcile == nil {
		t.Fatal("expected PayOSReconcile to be non-nil")
	}
	if res.PayOSReconcile.TotalTransactions != 100 {
		t.Errorf("expected TotalTransactions=100, got %d", res.PayOSReconcile.TotalTransactions)
	}
}

func TestGetRenewalStatsSuccessRateExcludesNonDecidedAttempts(t *testing.T) {
	repo := &mockAdminDashboardRepo{}
	reader := &mockSubscriptionReader{
		renewal: &subscriptionDTO.RenewalStatsDTO{
			TotalAttempts: 10,
			SuccessCount:  6,
			FailedCount:   2,
			AvgRetryCount: 1.5,
		},
	}

	uc := NewAdminDashboardUseCase(repo, &mockUserReader{}, &mockBrandStatsReader{}, &mockAICostReader{}, reader)
	res, err := uc.GetRenewalStats(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 6 succeeded of 8 decided (the other 2 are skipped/downgraded/renewed
	// outcomes, not failures): 75%, not the 60% the old total-attempts
	// denominator produced.
	if res.SuccessRate != 75.0 {
		t.Errorf("expected SuccessRate=75.0, got %v", res.SuccessRate)
	}
	if res.TotalAttempts != 10 || res.SuccessCount != 6 || res.FailedCount != 2 {
		t.Errorf("raw counts must pass through untouched: %+v", res)
	}
}

func TestGetPayOSReconciliationUsesRealBreakdown(t *testing.T) {
	repo := &mockAdminDashboardRepo{
		latestSnapshot: &entities.DashboardAdminDailySnapshot{
			PayOSTransactionCount: 100,
			PayOSSuccessCount:     90,
			PayOSPendingCount:     6,
			PayOSFailedCount:      2,
			PayOSVoidedCount:      2,
		},
	}

	uc := NewAdminDashboardUseCase(repo, &mockUserReader{}, &mockBrandStatsReader{}, &mockAICostReader{}, &mockSubscriptionReader{})
	res, err := uc.GetPayOSReconciliation(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The old formula reported failed = 100 - 90 = 10, lumping 6 pending and
	// 2 voided transactions into "failed".
	if res.FailedCount != 2 {
		t.Errorf("expected FailedCount=2, got %d", res.FailedCount)
	}
	if res.PendingCount != 6 || res.VoidedCount != 2 || res.SuccessCount != 90 {
		t.Errorf("breakdown mismatch: %+v", res)
	}
}
