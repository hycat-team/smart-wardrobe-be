package usecase

import (
	"context"
	"testing"
	"time"

	"smart-wardrobe-be/internal/modules/brand/domain/constants/brandmemberrole"
	"smart-wardrobe-be/internal/modules/dashboard/domain/entities"
	fashionDTO "smart-wardrobe-be/internal/modules/fashion/application/dto"
	fashionContract "smart-wardrobe-be/internal/modules/fashion/presentation/contract"
	wardrobeContract "smart-wardrobe-be/internal/modules/wardrobe/presentation/contract"

	brandDTO "smart-wardrobe-be/internal/modules/brand/application/dto"
	brandContract "smart-wardrobe-be/internal/modules/brand/presentation/contract"

	"github.com/google/uuid"
)

type mockBrandDashboardRepo struct {
	latestSnapshot      *entities.DashboardBrandDailySnapshot
	snapshots           []*entities.DashboardBrandDailySnapshot
	latestSnapshotError error
}

func (m *mockBrandDashboardRepo) UpsertAdminSnapshot(ctx context.Context, snapshot *entities.DashboardAdminDailySnapshot) error {
	return nil
}

func (m *mockBrandDashboardRepo) GetAdminSnapshots(ctx context.Context, fromDate, toDate time.Time) ([]*entities.DashboardAdminDailySnapshot, error) {
	return nil, nil
}

func (m *mockBrandDashboardRepo) GetLatestAdminSnapshot(ctx context.Context) (*entities.DashboardAdminDailySnapshot, error) {
	return nil, nil
}

func (m *mockBrandDashboardRepo) UpsertBrandSnapshot(ctx context.Context, snapshot *entities.DashboardBrandDailySnapshot) error {
	return nil
}

func (m *mockBrandDashboardRepo) GetBrandSnapshots(ctx context.Context, brandID uuid.UUID, fromDate, toDate time.Time) ([]*entities.DashboardBrandDailySnapshot, error) {
	return m.snapshots, nil
}

func (m *mockBrandDashboardRepo) GetLatestBrandSnapshot(ctx context.Context, brandID uuid.UUID) (*entities.DashboardBrandDailySnapshot, error) {
	if m.latestSnapshotError != nil {
		return nil, m.latestSnapshotError
	}
	return m.latestSnapshot, nil
}

func (m *mockBrandDashboardRepo) UpsertUserWardrobeSummary(ctx context.Context, summary *entities.DashboardUserWardrobeSummary) error {
	return nil
}

func (m *mockBrandDashboardRepo) GetUserWardrobeSummary(ctx context.Context, userID uuid.UUID) (*entities.DashboardUserWardrobeSummary, error) {
	return nil, nil
}

// Expansion stub methods
func (m *mockBrandDashboardRepo) GetSubscriptionDistribution(ctx context.Context) ([]map[string]interface{}, error) {
	return nil, nil
}
func (m *mockBrandDashboardRepo) GetRevenueBreakdown(ctx context.Context, fromDate, toDate time.Time) ([]map[string]interface{}, error) {
	return nil, nil
}
func (m *mockBrandDashboardRepo) GetAIUsageByOperation(ctx context.Context, fromDate, toDate time.Time) ([]map[string]interface{}, error) {
	return nil, nil
}
func (m *mockBrandDashboardRepo) GetAIErrorBreakdown(ctx context.Context) ([]map[string]interface{}, error) {
	return nil, nil
}
func (m *mockBrandDashboardRepo) GetRenewalStats(ctx context.Context) (map[string]interface{}, error) {
	return nil, nil
}
func (m *mockBrandDashboardRepo) GetCustomerAcquisition(ctx context.Context, brandID uuid.UUID) ([]map[string]interface{}, error) {
	return nil, nil
}
func (m *mockBrandDashboardRepo) GetLoyaltyTierDistribution(ctx context.Context, brandID uuid.UUID) ([]map[string]interface{}, error) {
	return nil, nil
}
func (m *mockBrandDashboardRepo) GetPointExpiryForecast(ctx context.Context, brandID uuid.UUID) ([]map[string]interface{}, error) {
	return nil, nil
}
func (m *mockBrandDashboardRepo) GetBenefitRedemptionAnalytics(ctx context.Context, brandID uuid.UUID) ([]map[string]interface{}, error) {
	return nil, nil
}
func (m *mockBrandDashboardRepo) GetCatalogStats(ctx context.Context, brandID uuid.UUID) (map[string]interface{}, error) {
	return nil, nil
}
func (m *mockBrandDashboardRepo) GetCustomerSpendSegmentation(ctx context.Context, brandID uuid.UUID) ([]map[string]interface{}, error) {
	return nil, nil
}
func (m *mockBrandDashboardRepo) GetWardrobeCategoryDistribution(ctx context.Context, userID uuid.UUID) ([]map[string]interface{}, error) {
	return nil, nil
}
func (m *mockBrandDashboardRepo) GetBrandNewCustomerCount(ctx context.Context, brandID uuid.UUID, fromDate, toDate time.Time) (float64, error) {
	return 0, nil
}
func (m *mockBrandDashboardRepo) GetBrandCustomerClaimRate(ctx context.Context, brandID uuid.UUID) (float64, error) {
	return 0, nil
}
func (m *mockBrandDashboardRepo) GetBrandRedemptionCount(ctx context.Context, brandID uuid.UUID, fromDate, toDate time.Time) (float64, error) {
	return 0, nil
}
func (m *mockBrandDashboardRepo) GetBrandItemCount(ctx context.Context, brandID uuid.UUID) (float64, error) {
	return 0, nil
}
func (m *mockBrandDashboardRepo) GetBrandItemCountByType(ctx context.Context, brandID uuid.UUID) ([]map[string]interface{}, error) {
	return nil, nil
}
func (m *mockBrandDashboardRepo) GetTotalAIRequestCount(ctx context.Context, fromDate, toDate time.Time) (int64, error) {
	return 0, nil
}

type mockBrandReader struct {
	brandContract.IBrandDashboardReader
	stats    *brandDTO.BrandAnalyticsStatsDTO
	role     brandmemberrole.BrandMemberRole
	roleErr  error
	statsErr error
}

func (m *mockBrandReader) GetBrandAnalyticsStats(ctx context.Context, brandID uuid.UUID, snapshotDate time.Time) (*brandDTO.BrandAnalyticsStatsDTO, error) {
	return m.stats, m.statsErr
}

func (m *mockBrandReader) GetUserBrandRole(ctx context.Context, userID uuid.UUID, brandID uuid.UUID) (brandmemberrole.BrandMemberRole, error) {
	return m.role, m.roleErr
}

type mockWardrobeReaderForBrand struct {
	wardrobeContract.IWardrobeDashboardReader
}

func (m *mockWardrobeReaderForBrand) GetOutfitItemFashionItemIDs(ctx context.Context, outfitIDs []uuid.UUID) ([]uuid.UUID, error) {
	return nil, nil
}

type mockFashionContractForBrand struct {
	fashionContract.IFashionContract
}

func (m *mockFashionContractForBrand) ListFashionItems(ctx context.Context, ids []uuid.UUID) ([]*fashionDTO.FashionItemRes, error) {
	return nil, nil
}

func TestBrandGetKPICards_FromSnapshot(t *testing.T) {
	brandID := uuid.New()
	repo := &mockBrandDashboardRepo{
		latestSnapshot: &entities.DashboardBrandDailySnapshot{
			BrandID:                   brandID,
			SnapshotDate:              time.Now().Truncate(24 * time.Hour),
			TotalMembers:              500,
			PointsIssued:              10000,
			PointsRedeemed:            3000,
			ActiveVouchers:            20,
			VouchersRedeemed:          8,
			DigitalSampleVotesCount:   150,
			DigitalSampleAvgRating:    4.5,
			OpenCSTickets:             10,
			CSFirstResponseAvgSeconds: 120,
		},
	}
	uc := NewBrandDashboardUseCase(repo, &mockBrandReader{}, &mockWardrobeReaderForBrand{}, &mockFashionContractForBrand{})
	res, err := uc.GetBrandKPICards(context.Background(), brandID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.TotalMembers != 500 {
		t.Errorf("expected TotalMembers=500, got %d", res.TotalMembers)
	}
	if res.DigitalSampleVotesCount != 150 {
		t.Errorf("expected DigitalSampleVotesCount=150, got %d", res.DigitalSampleVotesCount)
	}
	if res.DigitalSampleAvgRating != 4.5 {
		t.Errorf("expected DigitalSampleAvgRating=4.5, got %v", res.DigitalSampleAvgRating)
	}
	if res.OpenCSTickets != 10 {
		t.Errorf("expected OpenCSTickets=10, got %d", res.OpenCSTickets)
	}
	if res.CSFirstResponseAvgSeconds != 120 {
		t.Errorf("expected CSFirstResponseAvgSeconds=120, got %d", res.CSFirstResponseAvgSeconds)
	}
}

func TestBrandGetPointsLiability_OwnerSeesFinancial(t *testing.T) {
	brandID := uuid.New()
	userID := uuid.New()
	repo := &mockBrandDashboardRepo{
		latestSnapshot: &entities.DashboardBrandDailySnapshot{
			BrandID:                 brandID,
			PointsExpiring30d:       5000,
			PointsLiabilityValueVND: 5000000,
		},
	}
	reader := &mockBrandReader{
		role: brandmemberrole.Owner,
	}
	uc := NewBrandDashboardUseCase(repo, reader, &mockWardrobeReaderForBrand{}, &mockFashionContractForBrand{})
	res, err := uc.GetPointsLiability(context.Background(), userID, brandID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.PointsExpiring30d != 5000 {
		t.Errorf("expected PointsExpiring30d=5000, got %d", res.PointsExpiring30d)
	}
	if res.PointsLiabilityValueVND != 5000000 {
		t.Errorf("expected PointsLiabilityValueVND=5000000, got %v", res.PointsLiabilityValueVND)
	}
}

func TestBrandGetPointsLiability_StaffSeesRedacted(t *testing.T) {
	brandID := uuid.New()
	userID := uuid.New()
	repo := &mockBrandDashboardRepo{
		latestSnapshot: &entities.DashboardBrandDailySnapshot{
			BrandID:                 brandID,
			PointsExpiring30d:       5000,
			PointsLiabilityValueVND: 5000000,
		},
	}
	reader := &mockBrandReader{
		role: brandmemberrole.Staff,
	}
	uc := NewBrandDashboardUseCase(repo, reader, &mockWardrobeReaderForBrand{}, &mockFashionContractForBrand{})
	res, err := uc.GetPointsLiability(context.Background(), userID, brandID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.PointsExpiring30d != 5000 {
		t.Errorf("expected PointsExpiring30d=5000, got %d", res.PointsExpiring30d)
	}
	if res.PointsLiabilityValueVND != 0 {
		t.Errorf("expected PointsLiabilityValueVND=0 (redacted), got %v", res.PointsLiabilityValueVND)
	}
}
