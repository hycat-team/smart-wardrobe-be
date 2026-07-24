package brand

import (
	"context"
	"strings"
	"testing"
	"time"

	"smart-wardrobe-be/internal/modules/brand/application/dto"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/brandstatus"
	"smart-wardrobe-be/internal/modules/brand/domain/entities"
	"smart-wardrobe-be/internal/modules/brand/domain/repositories"

	"github.com/google/uuid"
)

type mockBrandRepo struct {
	brands map[uuid.UUID]*entities.Brand
}

func (m *mockBrandRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.Brand, error) {
	return m.brands[id], nil
}
func (m *mockBrandRepo) GetAll(ctx context.Context) ([]*entities.Brand, error)   { return nil, nil }
func (m *mockBrandRepo) Create(ctx context.Context, brand *entities.Brand) error { return nil }
func (m *mockBrandRepo) Update(ctx context.Context, brand *entities.Brand) error { return nil }
func (m *mockBrandRepo) Delete(ctx context.Context, id uuid.UUID) error          { return nil }
func (m *mockBrandRepo) GetBySlug(ctx context.Context, slug string) (*entities.Brand, error) {
	return nil, nil
}
func (m *mockBrandRepo) GetActive(ctx context.Context) ([]*entities.Brand, error) { return nil, nil }
func (m *mockBrandRepo) GetActiveFiltered(ctx context.Context, filter repositories.BrandFilter) (*repositories.BrandListResult, error) {
	activeStatus := brandstatus.Active
	filter.Status = &activeStatus
	return m.GetBrandsForAdmin(ctx, filter)
}
func (m *mockBrandRepo) GetBrandsForAdmin(ctx context.Context, filter repositories.BrandFilter) (*repositories.BrandListResult, error) {
	var list []*entities.Brand
	for _, b := range m.brands {
		if filter.Status != nil && b.Status != *filter.Status {
			continue
		}
		if filter.Query != nil {
			q := strings.ToLower(*filter.Query)
			if !strings.Contains(strings.ToLower(b.Name), q) && !strings.Contains(strings.ToLower(b.Slug), q) {
				continue
			}
		}
		list = append(list, b)
	}
	return &repositories.BrandListResult{
		Brands:     list,
		TotalCount: int64(len(list)),
	}, nil
}
func (m *mockBrandRepo) GetAdminBrandStats(ctx context.Context, snapshotDate time.Time) (*dto.BrandAdminStatsDTO, error) {
	return nil, nil
}
func (m *mockBrandRepo) GetBrandAnalyticsStats(ctx context.Context, brandID uuid.UUID, snapshotDate time.Time) (*dto.BrandAnalyticsStatsDTO, error) {
	return nil, nil
}
func (m *mockBrandRepo) GetSampleLabFeedbacksDrillDown(ctx context.Context, itemID uuid.UUID, page, limit int) ([]dto.SampleFeedbackDrillDownDTO, int64, error) {
	return nil, 0, nil
}

type mockCustomerRepo struct{}

func (m *mockCustomerRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.BrandCustomer, error) {
	return nil, nil
}
func (m *mockCustomerRepo) GetAll(ctx context.Context) ([]*entities.BrandCustomer, error) {
	return nil, nil
}
func (m *mockCustomerRepo) Create(ctx context.Context, customer *entities.BrandCustomer) error {
	return nil
}
func (m *mockCustomerRepo) Update(ctx context.Context, customer *entities.BrandCustomer) error {
	return nil
}
func (m *mockCustomerRepo) Delete(ctx context.Context, id uuid.UUID) error { return nil }
func (m *mockCustomerRepo) GetByBrandAndUser(ctx context.Context, brandID uuid.UUID, userID uuid.UUID) (*entities.BrandCustomer, error) {
	return nil, nil
}
func (m *mockCustomerRepo) GetByBrandAndPhoneHash(ctx context.Context, brandID uuid.UUID, phoneHash string) (*entities.BrandCustomer, error) {
	return nil, nil
}
func (m *mockCustomerRepo) GetByBrandAndExternalCode(ctx context.Context, brandID uuid.UUID, externalCustomerCode string) (*entities.BrandCustomer, error) {
	return nil, nil
}
func (m *mockCustomerRepo) GetByBrandID(ctx context.Context, brandID uuid.UUID) ([]*entities.BrandCustomer, error) {
	return nil, nil
}
func (m *mockCustomerRepo) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*entities.BrandCustomer, error) {
	return nil, nil
}
func (m *mockCustomerRepo) GetByBrandIDPaginated(ctx context.Context, filter repositories.BrandCustomerFilter) (*repositories.BrandCustomerListResult, error) {
	return nil, nil
}
func (m *mockCustomerRepo) CountByBrandID(ctx context.Context, brandID uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *mockCustomerRepo) CountByBrandIDs(ctx context.Context, brandIDs []uuid.UUID) (map[uuid.UUID]int64, error) {
	return map[uuid.UUID]int64{}, nil
}

func TestGetBrandsForAdmin(t *testing.T) {
	brandID1 := uuid.New()
	brandID2 := uuid.New()

	brandRepo := &mockBrandRepo{brands: map[uuid.UUID]*entities.Brand{
		brandID1: {
			Slug:   "closy",
			Name:   "Closy Brand",
			Status: brandstatus.Active,
		},
		brandID2: {
			Slug:   "gucci",
			Name:   "Gucci Brand",
			Status: brandstatus.PendingReview,
		},
	}}

	uc := &BrandUseCase{
		brandRepo:    brandRepo,
		customerRepo: &mockCustomerRepo{},
	}

	// 1. Get all brands (no filters)
	queryAll := dto.GetBrandsAdminQueryReq{}
	resAll, err := uc.GetBrandsForAdmin(context.Background(), queryAll)
	if err != nil {
		t.Fatalf("Expected nil err, got %v", err)
	}
	if len(resAll.Items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(resAll.Items))
	}
	if resAll.Metadata.TotalItems != 2 {
		t.Errorf("Expected TotalItems to be 2, got %d", resAll.Metadata.TotalItems)
	}

	// 2. Filter by status active
	activeStatus := brandstatus.Active
	queryActive := dto.GetBrandsAdminQueryReq{
		Status: &activeStatus,
	}
	resActive, err := uc.GetBrandsForAdmin(context.Background(), queryActive)
	if err != nil {
		t.Fatalf("Expected nil err, got %v", err)
	}
	if len(resActive.Items) != 1 {
		t.Errorf("Expected 1 item, got %d", len(resActive.Items))
	}
	if resActive.Items[0].Slug != "closy" {
		t.Errorf("Expected closy, got %s", resActive.Items[0].Slug)
	}

	// 3. Search query
	searchQueryStr := "Gucci"
	querySearch := dto.GetBrandsAdminQueryReq{
		Query: &searchQueryStr,
	}
	resSearch, err := uc.GetBrandsForAdmin(context.Background(), querySearch)
	if err != nil {
		t.Fatalf("Expected nil err, got %v", err)
	}
	if len(resSearch.Items) != 1 {
		t.Errorf("Expected 1 item, got %d", len(resSearch.Items))
	}
	if resSearch.Items[0].Slug != "gucci" {
		t.Errorf("Expected gucci, got %s", resSearch.Items[0].Slug)
	}
}

func TestGetActiveBrandsFiltersPublicList(t *testing.T) {
	brandID1 := uuid.New()
	brandID2 := uuid.New()
	brandID3 := uuid.New()

	brandRepo := &mockBrandRepo{brands: map[uuid.UUID]*entities.Brand{
		brandID1: {
			Slug:   "closy-active",
			Name:   "Closy Active Brand",
			Status: brandstatus.Active,
		},
		brandID2: {
			Slug:   "closy-pending",
			Name:   "Closy Pending Brand",
			Status: brandstatus.PendingReview,
		},
		brandID3: {
			Slug:   "other-active",
			Name:   "Other Active Brand",
			Status: brandstatus.Active,
		},
	}}

	uc := &BrandUseCase{
		brandRepo:    brandRepo,
		customerRepo: &mockCustomerRepo{},
	}

	searchQuery := "closy"
	res, err := uc.GetActiveBrands(context.Background(), dto.GetActiveBrandsQueryReq{
		Query: &searchQuery,
	})
	if err != nil {
		t.Fatalf("Expected nil err, got %v", err)
	}
	if len(res.Items) != 1 {
		t.Fatalf("Expected 1 item, got %d", len(res.Items))
	}
	if res.Items[0].Slug != "closy-active" {
		t.Errorf("Expected closy-active, got %s", res.Items[0].Slug)
	}
	if res.Metadata.TotalItems != 1 {
		t.Errorf("Expected TotalItems to be 1, got %d", res.Metadata.TotalItems)
	}
}
