package usecase

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	brandContract "smart-wardrobe-be/services/commerce/internal/modules/brand/presentation/contract"
	"smart-wardrobe-be/services/commerce/internal/modules/dashboard/domain/entities"
	fashionDTO "smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	fashionContract "smart-wardrobe-be/services/commerce/internal/modules/fashion/presentation/contract"
	wardrobeDTO "smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/dto"
	wardrobeContract "smart-wardrobe-be/services/commerce/internal/modules/wardrobe/presentation/contract"

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

func (m *mockUserDashboardRepo) UpsertUserWardrobeSummary(ctx context.Context, summary *entities.DashboardUserWardrobeSummary) error {
	m.summary = summary
	return nil
}

func (m *mockUserDashboardRepo) GetUserWardrobeSummary(ctx context.Context, userID uuid.UUID) (*entities.DashboardUserWardrobeSummary, error) {
	return m.summary, m.summaryErr
}

// Expansion stub methods
func (m *mockUserDashboardRepo) GetSubscriptionDistribution(ctx context.Context) ([]map[string]interface{}, error) {
	return nil, nil
}
func (m *mockUserDashboardRepo) GetRevenueBreakdown(ctx context.Context, fromDate, toDate time.Time) ([]map[string]interface{}, error) {
	return nil, nil
}
func (m *mockUserDashboardRepo) GetAIUsageByOperation(ctx context.Context, fromDate, toDate time.Time) ([]map[string]interface{}, error) {
	return nil, nil
}
func (m *mockUserDashboardRepo) GetAIErrorBreakdown(ctx context.Context) ([]map[string]interface{}, error) {
	return nil, nil
}
func (m *mockUserDashboardRepo) GetRenewalStats(ctx context.Context) (map[string]interface{}, error) {
	return nil, nil
}
func (m *mockUserDashboardRepo) GetCustomerAcquisition(ctx context.Context, brandID uuid.UUID) ([]map[string]interface{}, error) {
	return nil, nil
}
func (m *mockUserDashboardRepo) GetLoyaltyTierDistribution(ctx context.Context, brandID uuid.UUID) ([]map[string]interface{}, error) {
	return nil, nil
}
func (m *mockUserDashboardRepo) GetPointExpiryForecast(ctx context.Context, brandID uuid.UUID) ([]map[string]interface{}, error) {
	return nil, nil
}
func (m *mockUserDashboardRepo) GetBenefitRedemptionAnalytics(ctx context.Context, brandID uuid.UUID) ([]map[string]interface{}, error) {
	return nil, nil
}
func (m *mockUserDashboardRepo) GetCatalogStats(ctx context.Context, brandID uuid.UUID) (map[string]interface{}, error) {
	return nil, nil
}
func (m *mockUserDashboardRepo) GetCustomerSpendSegmentation(ctx context.Context, brandID uuid.UUID) ([]map[string]interface{}, error) {
	return nil, nil
}
func (m *mockUserDashboardRepo) GetWardrobeCategoryDistribution(ctx context.Context, userID uuid.UUID) ([]map[string]interface{}, error) {
	return nil, nil
}
func (m *mockUserDashboardRepo) GetBrandNewCustomerCount(ctx context.Context, brandID uuid.UUID, fromDate, toDate time.Time) (float64, error) {
	return 0, nil
}
func (m *mockUserDashboardRepo) GetBrandCustomerClaimRate(ctx context.Context, brandID uuid.UUID) (float64, error) {
	return 0, nil
}
func (m *mockUserDashboardRepo) GetBrandRedemptionCount(ctx context.Context, brandID uuid.UUID, fromDate, toDate time.Time) (float64, error) {
	return 0, nil
}
func (m *mockUserDashboardRepo) GetBrandItemCount(ctx context.Context, brandID uuid.UUID) (float64, error) {
	return 0, nil
}
func (m *mockUserDashboardRepo) GetBrandItemCountByType(ctx context.Context, brandID uuid.UUID) ([]map[string]interface{}, error) {
	return nil, nil
}
func (m *mockUserDashboardRepo) GetTotalAIRequestCount(ctx context.Context, fromDate, toDate time.Time) (int64, error) {
	return 0, nil
}

type mockWardrobeReader struct {
	wardrobeContract.IWardrobeDashboardReader
	stats                *wardrobeDTO.UserWardrobeSummaryStatsDTO
	underutilized        []*wardrobeDTO.UnderutilizedItemInfoDTO
	statsErr             error
	underutilizedErr     error
	fashionItemCounts    map[uuid.UUID]int64
	fashionItemCountsErr error
}

func (m *mockWardrobeReader) GetUserWardrobeSummaryStats(ctx context.Context, userID uuid.UUID) (*wardrobeDTO.UserWardrobeSummaryStatsDTO, error) {
	return m.stats, m.statsErr
}

func (m *mockWardrobeReader) GetUnderutilizedItems(ctx context.Context, userID uuid.UUID, limit int) ([]*wardrobeDTO.UnderutilizedItemInfoDTO, error) {
	return m.underutilized, m.underutilizedErr
}

func (m *mockWardrobeReader) GetUserWardrobeFashionItemCounts(ctx context.Context, userID uuid.UUID) (map[uuid.UUID]int64, error) {
	return m.fashionItemCounts, m.fashionItemCountsErr
}

type mockUserBrandReader struct {
	brandContract.IBrandDashboardReader
	expiringCount    int
	expiringCountErr error
}

func (m *mockUserBrandReader) GetUserExpiringBenefitsCount(ctx context.Context, userID uuid.UUID) (int, error) {
	return m.expiringCount, m.expiringCountErr
}

type mockFashionReaderForUser struct {
	fashionContract.IFashionDashboardReader
	mappings   []fashionDTO.DashboardFashionItemCategoryDTO
	categories []fashionDTO.DashboardCategoryDTO
}

func (m *mockFashionReaderForUser) GetCategoriesByIDs(ctx context.Context, categoryIDs []uuid.UUID) ([]fashionDTO.DashboardCategoryDTO, error) {
	return m.categories, nil
}

func (m *mockFashionReaderForUser) GetCategoryIDsByFashionItemIDs(ctx context.Context, fashionItemIDs []uuid.UUID) ([]fashionDTO.DashboardFashionItemCategoryDTO, error) {
	return m.mappings, nil
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
	uc := NewUserDashboardUseCase(repo, &mockWardrobeReader{}, &mockUserBrandReader{}, &mockFashionReaderForUser{})
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
	brandReader := &mockUserBrandReader{expiringCount: 5}
	uc := NewUserDashboardUseCase(repo, wardrobeReader, brandReader, &mockFashionReaderForUser{})
	res, err := uc.GetWardrobeInsights(context.Background(), userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.TotalItems != 20 {
		t.Errorf("expected TotalItems=20, got %d", res.TotalItems)
	}
	if len(res.UnderutilizedItems) != 1 {
		t.Errorf("expected 1 underutilized item, got %d", len(res.UnderutilizedItems))
	}
	if res.ExpiringVouchersCount != 5 {
		t.Errorf("expected ExpiringVouchersCount=5 (from brand reader), got %d", res.ExpiringVouchersCount)
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
	uc := NewUserDashboardUseCase(repo, &mockWardrobeReader{}, &mockUserBrandReader{}, &mockFashionReaderForUser{})
	res, err := uc.GetWardrobeInsights(context.Background(), userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.TotalItems != 0 || res.TotalWardrobeValueVND != 0 {
		t.Error("expected all fields to be 0 when no data")
	}
}

func TestGetWardrobeCategoryDistribution_CountsRowsWithMultiplicity(t *testing.T) {
	userID := uuid.New()
	fashionQuan := uuid.New()
	fashionAo := uuid.New()
	categoryQuan := uuid.New()
	categoryAo := uuid.New()

	wardrobeReader := &mockWardrobeReader{
		fashionItemCounts: map[uuid.UUID]int64{
			fashionQuan: 2,
			fashionAo:   1,
		},
	}
	fashionReader := &mockFashionReaderForUser{
		mappings: []fashionDTO.DashboardFashionItemCategoryDTO{
			{FashionItemID: fashionQuan, CategoryID: &categoryQuan},
			{FashionItemID: fashionAo, CategoryID: &categoryAo},
		},
		categories: []fashionDTO.DashboardCategoryDTO{
			{ID: categoryQuan, Name: "Quần"},
			{ID: categoryAo, Name: "Áo"},
		},
	}

	uc := NewUserDashboardUseCase(&mockUserDashboardRepo{}, wardrobeReader, &mockUserBrandReader{}, fashionReader)
	res, err := uc.GetWardrobeCategoryDistribution(context.Background(), userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.TotalItems != 3 {
		t.Errorf("expected TotalItems=3, got %d", res.TotalItems)
	}

	itemCountByCategory := map[uuid.UUID]int{}
	nameByCategory := map[uuid.UUID]string{}
	for _, c := range res.Categories {
		itemCountByCategory[c.CategoryID] = c.ItemCount
		nameByCategory[c.CategoryID] = c.CategoryName
	}
	if itemCountByCategory[categoryQuan] != 2 {
		t.Errorf("expected Quần itemCount=2 (duplicate fashion rows counted), got %d", itemCountByCategory[categoryQuan])
	}
	if itemCountByCategory[categoryAo] != 1 {
		t.Errorf("expected Áo itemCount=1, got %d", itemCountByCategory[categoryAo])
	}
	if nameByCategory[categoryQuan] != "Quần" {
		t.Errorf("expected category name Quần, got %q", nameByCategory[categoryQuan])
	}
}

func TestGetWardrobeCategoryDistribution_EmptyWardrobe(t *testing.T) {
	userID := uuid.New()
	uc := NewUserDashboardUseCase(&mockUserDashboardRepo{}, &mockWardrobeReader{}, &mockUserBrandReader{}, &mockFashionReaderForUser{})
	res, err := uc.GetWardrobeCategoryDistribution(context.Background(), userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.TotalItems != 0 || len(res.Categories) != 0 {
		t.Errorf("expected empty distribution, got total=%d categories=%d", res.TotalItems, len(res.Categories))
	}
}
