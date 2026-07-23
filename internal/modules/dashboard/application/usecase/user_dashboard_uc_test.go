package usecase

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"smart-wardrobe-be/internal/modules/dashboard/domain/entities"
	wardrobeDTO "smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	wardrobeContract "smart-wardrobe-be/internal/modules/wardrobe/presentation/contract"

	"github.com/google/uuid"
)

type mockUserDashboardRepo struct {
	summary    *entities.DashboardUserWardrobeSummary
	summaryErr error
}

func (m *mockUserDashboardRepo) UpsertAdminSnapshot(ctx context.Context, snapshot *entities.DashboardAdminDailySnapshot) error {
	return nil
}

func (m *mockUserDashboardRepo) GetAdminSnapshots(ctx context.Context, fromDate, toDate time.Time) ([]*entities.DashboardAdminDailySnapshot, error) {
	return nil, nil
}

func (m *mockUserDashboardRepo) GetLatestAdminSnapshot(ctx context.Context) (*entities.DashboardAdminDailySnapshot, error) {
	return nil, nil
}

func (m *mockUserDashboardRepo) UpsertBrandSnapshot(ctx context.Context, snapshot *entities.DashboardBrandDailySnapshot) error {
	return nil
}

func (m *mockUserDashboardRepo) GetBrandSnapshots(ctx context.Context, brandID uuid.UUID, fromDate, toDate time.Time) ([]*entities.DashboardBrandDailySnapshot, error) {
	return nil, nil
}

func (m *mockUserDashboardRepo) GetLatestBrandSnapshot(ctx context.Context, brandID uuid.UUID) (*entities.DashboardBrandDailySnapshot, error) {
	return nil, nil
}

func (m *mockUserDashboardRepo) UpsertBrandItemSnapshot(ctx context.Context, snapshot *entities.DashboardBrandItemDailySnapshot) error {
	return nil
}

func (m *mockUserDashboardRepo) GetLatestBrandItemSnapshot(ctx context.Context, brandID uuid.UUID, itemID uuid.UUID) (*entities.DashboardBrandItemDailySnapshot, error) {
	return nil, nil
}

func (m *mockUserDashboardRepo) UpsertUserWardrobeSummary(ctx context.Context, summary *entities.DashboardUserWardrobeSummary) error {
	m.summary = summary
	return nil
}

func (m *mockUserDashboardRepo) GetUserWardrobeSummary(ctx context.Context, userID uuid.UUID) (*entities.DashboardUserWardrobeSummary, error) {
	return m.summary, m.summaryErr
}

type mockWardrobeReader struct {
	wardrobeContract.IWardrobeDashboardReader
	stats            *wardrobeDTO.UserWardrobeSummaryStatsDTO
	underutilized    []*wardrobeDTO.UnderutilizedItemInfoDTO
	statsErr         error
	underutilizedErr error
}

func (m *mockWardrobeReader) GetUserWardrobeSummaryStats(ctx context.Context, userID uuid.UUID) (*wardrobeDTO.UserWardrobeSummaryStatsDTO, error) {
	return m.stats, m.statsErr
}

func (m *mockWardrobeReader) GetUnderutilizedItems(ctx context.Context, userID uuid.UUID, limit int) ([]*wardrobeDTO.UnderutilizedItemInfoDTO, error) {
	return m.underutilized, m.underutilizedErr
}

func TestUserGetWardrobeInsights_CacheHit(t *testing.T) {
	userID := uuid.New()
	underutilized := []wardrobeDTO.UnderutilizedItemInfoDTO{
		{ItemID: uuid.New(), Name: "Unused Jacket", WearCount: 0, CostPerWearVND: 500000, LastWornDaysAgo: 60},
	}
	underutilizedJSON, _ := json.Marshal(underutilized)

	repo := &mockUserDashboardRepo{
		summary: &entities.DashboardUserWardrobeSummary{
			UserID:                  userID,
			TotalItems:              10,
			TotalWardrobeValueVND:   5000000,
			AverageCostPerWearVND:   50000,
			UnderutilizedItemsCount: 1,
			UnderutilizedItems:      underutilizedJSON,
			AIQuotaRemaining:        8,
			ExpiringVouchersCount:   3,
			UpdatedAt:               time.Now(),
		},
	}
	uc := NewUserDashboardUseCase(repo, &mockWardrobeReader{})
	res, err := uc.GetWardrobeInsights(context.Background(), userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.TotalItems != 10 {
		t.Errorf("expected TotalItems=10, got %d", res.TotalItems)
	}
	if len(res.UnderutilizedItems) != 1 {
		t.Errorf("expected 1 underutilized item, got %d", len(res.UnderutilizedItems))
	}
	if res.AIQuotaRemaining != 8 {
		t.Errorf("expected AIQuotaRemaining=8, got %d", res.AIQuotaRemaining)
	}
	if res.ExpiringVouchersCount != 3 {
		t.Errorf("expected ExpiringVouchersCount=3, got %d", res.ExpiringVouchersCount)
	}
}

func TestUserGetWardrobeInsights_CacheMissOnDemand(t *testing.T) {
	userID := uuid.New()
	repo := &mockUserDashboardRepo{
		summary: nil,
	}
	wardrobeReader := &mockWardrobeReader{
		stats: &wardrobeDTO.UserWardrobeSummaryStatsDTO{
			UserID:                  userID,
			TotalItems:              20,
			TotalWardrobeValueVND:   10000000,
			AverageCostPerWearVND:   25000,
			UnderutilizedItemsCount: 3,
		},
		underutilized: []*wardrobeDTO.UnderutilizedItemInfoDTO{
			{ItemID: uuid.New(), Name: "Rarely Worn Dress", WearCount: 1, CostPerWearVND: 800000, LastWornDaysAgo: 45},
		},
	}
	uc := NewUserDashboardUseCase(repo, wardrobeReader)
	res, err := uc.GetWardrobeInsights(context.Background(), userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.TotalItems != 20 {
		t.Errorf("expected TotalItems=20, got %d", res.TotalItems)
	}
	if res.AverageItemValueVND != 25000 {
		t.Errorf("expected AverageItemValueVND=25000, got %v", res.AverageItemValueVND)
	}
	if len(res.UnderutilizedItems) != 1 {
		t.Errorf("expected 1 underutilized item, got %d", len(res.UnderutilizedItems))
	}
	if res.AIQuotaRemaining != 0 {
		t.Errorf("expected AIQuotaRemaining=0 (no stale snapshot), got %d", res.AIQuotaRemaining)
	}
	if res.ExpiringVouchersCount != 0 {
		t.Errorf("expected ExpiringVouchersCount=0 (no stale snapshot), got %d", res.ExpiringVouchersCount)
	}
	// Verify write-back was called (snapshot upserted)
	if repo.summary == nil {
		t.Fatal("expected snapshot to be upserted after cache miss")
	}
	if repo.summary.TotalItems != 20 {
		t.Errorf("expected upserted TotalItems=20, got %d", repo.summary.TotalItems)
	}
}

func TestUserGetWardrobeInsights_NoData(t *testing.T) {
	userID := uuid.New()
	repo := &mockUserDashboardRepo{}
	uc := NewUserDashboardUseCase(repo, &mockWardrobeReader{})
	_, err := uc.GetWardrobeInsights(context.Background(), userID)
	if err == nil {
		t.Fatal("expected error but got nil")
	}
}
