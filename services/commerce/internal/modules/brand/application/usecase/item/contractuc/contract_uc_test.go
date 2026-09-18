package contractuc_test

import (
	"context"
	"testing"

	uc_interfaces "smart-wardrobe-be/services/commerce/internal/modules/brand/application/interface/usecase"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/application/usecase/item/contractuc"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/constants/brandcustomerstatus"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/constants/branditem/branditemstatus"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/constants/brandstatus"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/entities"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/repositories"

	"github.com/google/uuid"
)

type mockBrandRepo struct {
	repositories.IBrandRepository
	getByIDFn func(ctx context.Context, id uuid.UUID) (*entities.Brand, error)
}

func (m *mockBrandRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.Brand, error) {
	return m.getByIDFn(ctx, id)
}

type mockCustomerRepo struct {
	repositories.IBrandCustomerRepository
	getByBrandAndUserFn func(ctx context.Context, brandID uuid.UUID, userID uuid.UUID) (*entities.BrandCustomer, error)
}

func (m *mockCustomerRepo) GetByBrandAndUser(ctx context.Context, brandID uuid.UUID, userID uuid.UUID) (*entities.BrandCustomer, error) {
	return m.getByBrandAndUserFn(ctx, brandID, userID)
}

type mockBrandItemRepo struct {
	repositories.IBrandItemRepository
	getByFashionItemIDFn func(ctx context.Context, id uuid.UUID) (*entities.BrandItem, error)
}

func (m *mockBrandItemRepo) GetByFashionItemID(ctx context.Context, id uuid.UUID) (*entities.BrandItem, error) {
	return m.getByFashionItemIDFn(ctx, id)
}

type mockBenefitUC struct {
	uc_interfaces.IBrandBenefitUseCase
}

func TestCheckBrandItemEligibility_ReturnsPureBrandItem(t *testing.T) {
	userID := uuid.New()
	fashionItemID := uuid.New()
	brandID := uuid.New()

	brandRepo := &mockBrandRepo{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*entities.Brand, error) {
			b := &entities.Brand{
				Status: brandstatus.Active,
			}
			b.ID = brandID
			return b, nil
		},
	}

	customerRepo := &mockCustomerRepo{
		getByBrandAndUserFn: func(ctx context.Context, bID uuid.UUID, uID uuid.UUID) (*entities.BrandCustomer, error) {
			return &entities.BrandCustomer{
				BrandID: brandID,
				UserID:  &userID,
				Status:  brandcustomerstatus.Active,
			}, nil
		},
	}

	brandItemRepo := &mockBrandItemRepo{
		getByFashionItemIDFn: func(ctx context.Context, fID uuid.UUID) (*entities.BrandItem, error) {
			bi := &entities.BrandItem{
				BrandID:       brandID,
				FashionItemID: &fashionItemID,
				Status:        branditemstatus.Active,
			}
			bi.ID = uuid.New()
			return bi, nil
		},
	}

	benefitUC := &mockBenefitUC{}

	uc := contractuc.NewBrandItemContractUseCase(brandRepo, customerRepo, brandItemRepo, benefitUC)
	eligible, res, err := uc.CheckBrandItemEligibility(context.Background(), userID, fashionItemID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !eligible {
		t.Fatalf("expected brand item to be eligible")
	}

	if res.FashionItemID == uuid.Nil || res.FashionItemID != fashionItemID {
		t.Errorf("expected FashionItemID to be %v, got %v", fashionItemID, res.FashionItemID)
	}
}
