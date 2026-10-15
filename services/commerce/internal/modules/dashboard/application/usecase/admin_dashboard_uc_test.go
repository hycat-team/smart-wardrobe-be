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
	"smart-wardrobe-be/shared/pkg/utils/timeutils"

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

	usageByOperation []aiDTO.AIUsageByOperationDTO
	errorBreakdown   []aiDTO.AIErrorBreakdownDTO
	totalRequests    int64

	errorBreakdownCalls int
	requestCountCalls   int
	lastErrorFrom       time.Time
	lastErrorTo         time.Time
	lastRequestFrom     time.Time
	lastRequestTo       time.Time
	lastUsageFrom       time.Time
	lastUsageTo         time.Time
}

func (m *mockAICostReader) GetAdminAIUsageStats(ctx context.Context, snapshotDate time.Time) (*aiDTO.AIUsageAdminStatsDTO, error) {
	return m.stats, m.err
}

func (m *mockAICostReader) GetAIEventsDrillDown(ctx context.Context, status string, page, limit int) ([]aiDTO.AIEventDrillDownDTO, int64, error) {
	return nil, 0, nil
}

func (m *mockAICostReader) GetAIUsageByOperation(ctx context.Context, fromDate, toDate time.Time) ([]aiDTO.AIUsageByOperationDTO, error) {
	m.lastUsageFrom, m.lastUsageTo = fromDate, toDate
	return m.usageByOperation, m.err
}

func (m *mockAICostReader) GetAIErrorBreakdown(ctx context.Context, from, to time.Time) ([]aiDTO.AIErrorBreakdownDTO, error) {
	m.errorBreakdownCalls++
	m.lastErrorFrom, m.lastErrorTo = from, to
	return m.errorBreakdown, m.err
}

func (m *mockAICostReader) GetTotalAIRequestCount(ctx context.Context, from, to time.Time) (int64, error) {
	m.requestCountCalls++
	m.lastRequestFrom, m.lastRequestTo = from, to
	return m.totalRequests, m.err
}

type mockSubscriptionReader struct {
	subscriptionContract.ISubscriptionDashboardReader
	stats        *subscriptionDTO.AdminSubscriptionStatsDTO
	renewal      *subscriptionDTO.RenewalStatsDTO
	distribution []subscriptionDTO.SubscriptionDistributionDTO
	revenue      []subscriptionDTO.RevenueBreakdownDTO
	err          error

	statsCalls           int
	lastStatsFrom        time.Time
	lastStatsTo          time.Time
	lastRenewalFrom      time.Time
	lastRenewalTo        time.Time
	lastDistributionFrom time.Time
	lastDistributionTo   time.Time
	lastRevenueFrom      time.Time
	lastRevenueTo        time.Time
}

func (m *mockSubscriptionReader) GetAdminSubscriptionStats(ctx context.Context, from, to time.Time) (*subscriptionDTO.AdminSubscriptionStatsDTO, error) {
	m.statsCalls++
	m.lastStatsFrom, m.lastStatsTo = from, to
	return m.stats, m.err
}

func (m *mockSubscriptionReader) GetSubscriptionDistribution(ctx context.Context, from, to time.Time) ([]subscriptionDTO.SubscriptionDistributionDTO, error) {
	m.lastDistributionFrom, m.lastDistributionTo = from, to
	return m.distribution, m.err
}

func (m *mockSubscriptionReader) GetRenewalStats(ctx context.Context, from, to time.Time) (*subscriptionDTO.RenewalStatsDTO, error) {
	m.lastRenewalFrom, m.lastRenewalTo = from, to
	return m.renewal, m.err
}

func (m *mockSubscriptionReader) GetRevenueBreakdown(ctx context.Context, from, to time.Time) ([]subscriptionDTO.RevenueBreakdownDTO, error) {
	m.lastRevenueFrom, m.lastRevenueTo = from, to
	return m.revenue, m.err
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

// 029 fixup: the margin response echoes the VN calendar days — 00:00 of the
// first and of the last day — never the exclusive upper bound (00:00 of the
// day AFTER the last one), which is what the raw EndOfDayInVN bound would
// print.
func TestAdminGetAIMarginAnalytics_EchoesVNDayLabels(t *testing.T) {
	uc := NewAdminDashboardUseCase(&mockAdminDashboardRepo{}, &mockUserReader{}, &mockBrandStatsReader{}, &mockAICostReader{}, &mockSubscriptionReader{})

	from := time.Date(2026, time.March, 1, 0, 0, 0, 0, timeutils.VnLoc())
	to := time.Date(2026, time.April, 1, 0, 0, 0, 0, timeutils.VnLoc()) // exclusive bound; last day is Mar 31

	res, err := uc.GetAIMarginAnalytics(context.Background(), from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wantTo := time.Date(2026, time.March, 31, 0, 0, 0, 0, timeutils.VnLoc())
	if !res.FromDate.Equal(from) {
		t.Errorf("expected FromDate=%v, got %v", from, res.FromDate)
	}
	if !res.ToDate.Equal(wantTo) {
		t.Errorf("expected ToDate=%v, got %v", wantTo, res.ToDate)
	}
}

func TestAdminGetPayOSReconciliation(t *testing.T) {
	// 029 T027/R6: the payOS panel now aggregates the transaction table over a
	// period instead of reading the latest snapshot; the counts come from the
	// subscription reader and the absent period defaults to today (VN).
	reader := &mockSubscriptionReader{
		stats: &subscriptionDTO.AdminSubscriptionStatsDTO{
			PayOSTransactionCount:    100,
			PayOSSuccessCount:        95,
			PayOSPendingCount:        3,
			PayOSFailedCount:         1,
			PayOSVoidedCount:         1,
			TotalSubscriptionRevenue: decimal.NewFromInt(5000000),
		},
	}
	uc := NewAdminDashboardUseCase(&mockAdminDashboardRepo{}, &mockUserReader{}, &mockBrandStatsReader{}, &mockAICostReader{}, reader)
	res, err := uc.GetPayOSReconciliation(context.Background(), time.Time{}, time.Time{})
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

	// No period sent: the reader must be asked for exactly today's VN day.
	wantFrom := timeutils.StartOfDayInVN(timeutils.NowVN())
	wantTo := timeutils.EndOfDayInVN(timeutils.NowVN())
	if !reader.lastStatsFrom.Equal(wantFrom) || !reader.lastStatsTo.Equal(wantTo) {
		t.Errorf("default period = [%s, %s), want today's VN day [%s, %s)",
			reader.lastStatsFrom, reader.lastStatsTo, wantFrom, wantTo)
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
	// 029 T027/T035: `overview` passes zero bounds to the payOS reconciliation
	// and the use case fills in today (VN) — the panel never reads a snapshot.
	reader := &mockSubscriptionReader{
		stats: &subscriptionDTO.AdminSubscriptionStatsDTO{
			PayOSTransactionCount: 100,
			PayOSSuccessCount:     95,
		},
	}
	uc := NewAdminDashboardUseCase(repo, &mockUserReader{}, &mockBrandStatsReader{}, &mockAICostReader{}, reader)
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
	if reader.statsCalls == 0 {
		t.Error("overview must reach the subscription reader for the payOS panel")
	}
	wantFrom := timeutils.StartOfDayInVN(timeutils.NowVN())
	if !reader.lastStatsFrom.Equal(wantFrom) {
		t.Errorf("overview payOS period must default to today VN, got from=%s want %s", reader.lastStatsFrom, wantFrom)
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
	// 029 T027: distribution/renewal pass the bounds straight through; zero
	// means "no period" and that meaning lives in the account service.
	res, err := uc.GetRenewalStats(context.Background(), time.Time{}, time.Time{})
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
	if !reader.lastRenewalFrom.IsZero() || !reader.lastRenewalTo.IsZero() {
		t.Errorf("an absent period must stay zero all the way to the reader, got [%s, %s)",
			reader.lastRenewalFrom, reader.lastRenewalTo)
	}
}

func TestGetPayOSReconciliationUsesRealBreakdown(t *testing.T) {
	reader := &mockSubscriptionReader{
		stats: &subscriptionDTO.AdminSubscriptionStatsDTO{
			PayOSTransactionCount: 100,
			PayOSSuccessCount:     90,
			PayOSPendingCount:     6,
			PayOSFailedCount:      2,
			PayOSVoidedCount:      2,
		},
	}

	uc := NewAdminDashboardUseCase(&mockAdminDashboardRepo{}, &mockUserReader{}, &mockBrandStatsReader{}, &mockAICostReader{}, reader)
	res, err := uc.GetPayOSReconciliation(context.Background(), time.Time{}, time.Time{})
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

// ---- 029 T027: the period is honoured everywhere ---------------------------------

// TestAdminGetPayOSReconciliationUsesTheRequestedPeriod is FR-017's "tổng hợp
// theo kỳ" half: an explicit period must reach the reader unchanged, not be
// replaced by a snapshot or by today.
func TestAdminGetPayOSReconciliationUsesTheRequestedPeriod(t *testing.T) {
	reader := &mockSubscriptionReader{
		stats: &subscriptionDTO.AdminSubscriptionStatsDTO{PayOSTransactionCount: 7},
	}
	uc := NewAdminDashboardUseCase(&mockAdminDashboardRepo{}, &mockUserReader{}, &mockBrandStatsReader{}, &mockAICostReader{}, reader)

	from := time.Date(2026, 9, 16, 0, 0, 0, 0, timeutils.VnLoc())
	to := time.Date(2026, 10, 16, 0, 0, 0, 0, timeutils.VnLoc())
	if _, err := uc.GetPayOSReconciliation(context.Background(), from, to); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !reader.lastStatsFrom.Equal(from) || !reader.lastStatsTo.Equal(to) {
		t.Errorf("reader period = [%s, %s), want [%s, %s)", reader.lastStatsFrom, reader.lastStatsTo, from, to)
	}
}

// TestAdminGetSubscriptionDistributionPassesThePeriodThrough covers the
// distribution signature: the payload semantics live in the account service
// (R7), so commerce must forward the bounds untouched — including zero for an
// absent period.
func TestAdminGetSubscriptionDistributionPassesThePeriodThrough(t *testing.T) {
	reader := &mockSubscriptionReader{
		distribution: []subscriptionDTO.SubscriptionDistributionDTO{
			{PlanCode: "FREE", UserCount: 3},
			{PlanCode: "PREMIUM_MONTHLY", UserCount: 1},
		},
	}
	uc := NewAdminDashboardUseCase(&mockAdminDashboardRepo{}, &mockUserReader{}, &mockBrandStatsReader{}, &mockAICostReader{}, reader)

	from := time.Date(2026, 10, 1, 0, 0, 0, 0, timeutils.VnLoc())
	to := time.Date(2026, 11, 1, 0, 0, 0, 0, timeutils.VnLoc())
	res, err := uc.GetSubscriptionDistribution(context.Background(), from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reader.lastDistributionFrom.Equal(from) || !reader.lastDistributionTo.Equal(to) {
		t.Errorf("reader period = [%s, %s), want [%s, %s)",
			reader.lastDistributionFrom, reader.lastDistributionTo, from, to)
	}
	if res.TotalUsers != 4 {
		t.Errorf("expected TotalUsers=4, got %d", res.TotalUsers)
	}
	for _, d := range res.Distribution {
		if d.PlanCode == "PREMIUM_MONTHLY" && d.Percentage != 25 {
			t.Errorf("expected the premium share to be 25%%, got %v", d.Percentage)
		}
	}

	// No period: both bounds stay zero so the account service keeps its
	// current-state semantics.
	if _, err := uc.GetSubscriptionDistribution(context.Background(), time.Time{}, time.Time{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reader.lastDistributionFrom.IsZero() || !reader.lastDistributionTo.IsZero() {
		t.Errorf("an absent period must stay zero, got [%s, %s)", reader.lastDistributionFrom, reader.lastDistributionTo)
	}
}

// TestAdminGetAIErrorBreakdownSharesOnePeriod is FR-016's core: numerator,
// denominator and grouping all use the SAME period, so the rate cannot exceed
// 100% while E <= R, and it is 0 when the sample is empty.
func TestAdminGetAIErrorBreakdownSharesOnePeriod(t *testing.T) {
	reader := &mockAICostReader{
		errorBreakdown: []aiDTO.AIErrorBreakdownDTO{
			{ErrorCode: "provider_timeout", ErrorCount: 2},
			{ErrorCode: "usage_unreported", ErrorCount: 3},
		},
		totalRequests: 20,
	}
	uc := NewAdminDashboardUseCase(&mockAdminDashboardRepo{}, &mockUserReader{}, &mockBrandStatsReader{}, reader, &mockSubscriptionReader{})

	from := time.Date(2026, 9, 16, 0, 0, 0, 0, timeutils.VnLoc())
	to := time.Date(2026, 10, 16, 0, 0, 0, 0, timeutils.VnLoc())
	res, err := uc.GetAIErrorBreakdown(context.Background(), from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.TotalErrors != 5 {
		t.Errorf("totalErrors must be the sum of the breakdown (2+3), got %d", res.TotalErrors)
	}
	if res.TotalRequests != 20 {
		t.Errorf("expected TotalRequests=20, got %d", res.TotalRequests)
	}
	if res.ErrorRate != 25.0 {
		t.Errorf("expected ErrorRate=5/20*100=25, got %v", res.ErrorRate)
	}
	if res.ErrorRate > 100 {
		t.Errorf("FR-016: an error rate with E <= R can never exceed 100%%, got %v", res.ErrorRate)
	}
	if !reader.lastErrorFrom.Equal(from) || !reader.lastErrorTo.Equal(to) {
		t.Errorf("breakdown period = [%s, %s), want [%s, %s)", reader.lastErrorFrom, reader.lastErrorTo, from, to)
	}
	if !reader.lastRequestFrom.Equal(from) || !reader.lastRequestTo.Equal(to) {
		t.Errorf("request-count period = [%s, %s), want the SAME [%s, %s)",
			reader.lastRequestFrom, reader.lastRequestTo, from, to)
	}

	// Empty sample: the rate is defined as 0, never a division by zero.
	reader.totalRequests = 0
	res, err = uc.GetAIErrorBreakdown(context.Background(), from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ErrorRate != 0 {
		t.Errorf("an empty sample must produce 0%%, got %v", res.ErrorRate)
	}
}

// TestAdminGetAIErrorBreakdownDefaultsToThirtyVNDays keeps the panel's
// backward-compatible default (FR-012/FR-016) while moving it onto VN
// calendar-day boundaries.
func TestAdminGetAIErrorBreakdownDefaultsToThirtyVNDays(t *testing.T) {
	reader := &mockAICostReader{}
	uc := NewAdminDashboardUseCase(&mockAdminDashboardRepo{}, &mockUserReader{}, &mockBrandStatsReader{}, reader, &mockSubscriptionReader{})

	if _, err := uc.GetAIErrorBreakdown(context.Background(), time.Time{}, time.Time{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	now := timeutils.NowVN()
	wantFrom := timeutils.StartOfDayInVN(now.AddDate(0, 0, -30))
	wantTo := timeutils.EndOfDayInVN(now)
	if !reader.lastErrorFrom.Equal(wantFrom) || !reader.lastErrorTo.Equal(wantTo) {
		t.Errorf("default breakdown period = [%s, %s), want [%s, %s)",
			reader.lastErrorFrom, reader.lastErrorTo, wantFrom, wantTo)
	}
	if !reader.lastRequestFrom.Equal(wantFrom) || !reader.lastRequestTo.Equal(wantTo) {
		t.Errorf("the request count must default to the same period, got [%s, %s)",
			reader.lastRequestFrom, reader.lastRequestTo)
	}
}

// TestAdminGetRevenueBreakdownDefaultsToSixVNDayBounds pins the intentional
// FR-012 adjustment: the 6-month default survives, but its edges are now VN
// calendar-day boundaries.
func TestAdminGetRevenueBreakdownDefaultsToSixVNDayBounds(t *testing.T) {
	reader := &mockSubscriptionReader{}
	uc := NewAdminDashboardUseCase(&mockAdminDashboardRepo{}, &mockUserReader{}, &mockBrandStatsReader{}, &mockAICostReader{}, reader)

	if _, err := uc.GetRevenueBreakdown(context.Background(), time.Time{}, time.Time{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	now := timeutils.NowVN()
	wantFrom := timeutils.StartOfDayInVN(now.AddDate(0, -6, 0))
	wantTo := timeutils.EndOfDayInVN(now)
	if !reader.lastRevenueFrom.Equal(wantFrom) || !reader.lastRevenueTo.Equal(wantTo) {
		t.Errorf("default revenue period = [%s, %s), want [%s, %s)",
			reader.lastRevenueFrom, reader.lastRevenueTo, wantFrom, wantTo)
	}
}

// TestAdminGetAIUsageByOperationDefaultsToThirtyVNDays mirrors the same
// boundary move for the other pre-existing period endpoint.
func TestAdminGetAIUsageByOperationDefaultsToThirtyVNDays(t *testing.T) {
	reader := &mockAICostReader{}
	uc := NewAdminDashboardUseCase(&mockAdminDashboardRepo{}, &mockUserReader{}, &mockBrandStatsReader{}, reader, &mockSubscriptionReader{})

	if _, err := uc.GetAIUsageByOperation(context.Background(), time.Time{}, time.Time{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	now := timeutils.NowVN()
	wantFrom := timeutils.StartOfDayInVN(now.AddDate(0, 0, -30))
	wantTo := timeutils.EndOfDayInVN(now)
	if !reader.lastUsageFrom.Equal(wantFrom) || !reader.lastUsageTo.Equal(wantTo) {
		t.Errorf("default usage period = [%s, %s), want [%s, %s)",
			reader.lastUsageFrom, reader.lastUsageTo, wantFrom, wantTo)
	}
}
