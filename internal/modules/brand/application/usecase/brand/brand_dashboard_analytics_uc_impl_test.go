package brand

import (
	"context"
	"testing"
	"time"

	"smart-wardrobe-be/internal/modules/brand/domain/repositories"

	"github.com/google/uuid"
)

type mockCustomerRepoForAnalytics struct {
	repositories.IBrandCustomerRepository
	sources  []repositories.CustomerSourceCount
	newCount int64
	claimed  int64
	total    int64
}

func (m *mockCustomerRepoForAnalytics) CountByJoinedSource(ctx context.Context, brandID uuid.UUID) ([]repositories.CustomerSourceCount, error) {
	return m.sources, nil
}

func (m *mockCustomerRepoForAnalytics) CountNewByBrandID(ctx context.Context, brandID uuid.UUID, fromDate, toDate time.Time) (int64, error) {
	return m.newCount, nil
}

func (m *mockCustomerRepoForAnalytics) CountClaimedByBrandID(ctx context.Context, brandID uuid.UUID) (int64, error) {
	return m.claimed, nil
}

func (m *mockCustomerRepoForAnalytics) CountByBrandID(ctx context.Context, brandID uuid.UUID) (int64, error) {
	return m.total, nil
}

type mockLoyaltyRepoForAnalytics struct {
	repositories.ILoyaltyAccountRepository
	tiers    []repositories.TierMemberCount
	segments []repositories.SpendSegmentCount
}

func (m *mockLoyaltyRepoForAnalytics) CountByTier(ctx context.Context, brandID uuid.UUID) ([]repositories.TierMemberCount, error) {
	return m.tiers, nil
}

func (m *mockLoyaltyRepoForAnalytics) CountBySpendSegment(ctx context.Context, brandID uuid.UUID) ([]repositories.SpendSegmentCount, error) {
	return m.segments, nil
}

type mockBrandItemRepoForAnalytics struct {
	repositories.IBrandItemRepository
	byStatus []repositories.BrandItemStatusCount
	byType   []repositories.BrandItemTypeCount
	total    int64
}

func (m *mockBrandItemRepoForAnalytics) CountByStatus(ctx context.Context, brandID uuid.UUID) ([]repositories.BrandItemStatusCount, error) {
	return m.byStatus, nil
}

func (m *mockBrandItemRepoForAnalytics) CountByType(ctx context.Context, brandID uuid.UUID) ([]repositories.BrandItemTypeCount, error) {
	return m.byType, nil
}

func (m *mockBrandItemRepoForAnalytics) CountByBrandID(ctx context.Context, brandID uuid.UUID) (int64, error) {
	return m.total, nil
}

func TestGetCustomerAcquisition(t *testing.T) {
	brandID := uuid.New()
	uc := &BrandDashboardAnalyticsUseCase{
		customerRepo: &mockCustomerRepoForAnalytics{
			sources: []repositories.CustomerSourceCount{
				{Source: "manual", CustomerCount: 30},
				{Source: "online", CustomerCount: 70},
			},
		},
	}

	res, err := uc.GetCustomerAcquisition(context.Background(), brandID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res) != 2 {
		t.Fatalf("expected 2 sources, got %d", len(res))
	}
	if res[0].Source != "manual" || res[0].CustomerCount != 30 {
		t.Errorf("unexpected first source: %+v", res[0])
	}
}

func TestGetCatalogStats(t *testing.T) {
	brandID := uuid.New()
	uc := &BrandDashboardAnalyticsUseCase{
		brandItemRepo: &mockBrandItemRepoForAnalytics{
			byStatus: []repositories.BrandItemStatusCount{{Status: "active", Count: 10}},
			byType:   []repositories.BrandItemTypeCount{{ItemType: "ao", Count: 6}},
			total:    10,
		},
	}

	res, err := uc.GetCatalogStats(context.Background(), brandID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.TotalCount != 10 {
		t.Errorf("expected total 10, got %d", res.TotalCount)
	}
	if len(res.ByStatus) != 1 || res.ByStatus[0].Label != "active" {
		t.Errorf("unexpected byStatus: %+v", res.ByStatus)
	}
	if len(res.ByType) != 1 || res.ByType[0].Label != "ao" {
		t.Errorf("unexpected byType: %+v", res.ByType)
	}
}

func TestGetLoyaltyTierDistribution(t *testing.T) {
	brandID := uuid.New()
	tierID := uuid.New()
	uc := &BrandDashboardAnalyticsUseCase{
		loyaltyRepo: &mockLoyaltyRepoForAnalytics{
			tiers: []repositories.TierMemberCount{
				{TierID: tierID, TierName: "Gold", TierRank: 1, MemberCount: 25},
			},
		},
	}

	res, err := uc.GetLoyaltyTierDistribution(context.Background(), brandID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res) != 1 {
		t.Fatalf("expected 1 tier, got %d", len(res))
	}
	if res[0].TierID != tierID || res[0].MemberCount != 25 {
		t.Errorf("unexpected tier: %+v", res[0])
	}
}

func TestGetBrandCustomerClaimRate(t *testing.T) {
	brandID := uuid.New()
	uc := &BrandDashboardAnalyticsUseCase{
		customerRepo: &mockCustomerRepoForAnalytics{total: 100, claimed: 40},
	}

	rate, err := uc.GetBrandCustomerClaimRate(context.Background(), brandID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rate != 40.0 {
		t.Errorf("expected claim rate 40, got %v", rate)
	}
}

func TestGetBrandNewCustomerCount(t *testing.T) {
	brandID := uuid.New()
	uc := &BrandDashboardAnalyticsUseCase{
		customerRepo: &mockCustomerRepoForAnalytics{newCount: 5},
	}

	count, err := uc.GetBrandNewCustomerCount(context.Background(), brandID, time.Now().AddDate(0, 0, -7), time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 5 {
		t.Errorf("expected 5, got %d", count)
	}
}
