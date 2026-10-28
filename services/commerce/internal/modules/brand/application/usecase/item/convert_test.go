package item

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/constants/branditem/branditemstatus"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/constants/branditem/branditemtype"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/entities"
	fashion_dto "smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/shared/application/constants/apperror"

	"github.com/google/uuid"
)

type mockFashionContractWithCategory struct {
	mockFashionContract
	category *fashion_dto.CategoryBriefRes
}

func (m *mockFashionContractWithCategory) GetFashionItem(ctx context.Context, id uuid.UUID) (*fashion_dto.FashionItemContractDTO, error) {
	return &fashion_dto.FashionItemContractDTO{
		ID:       id,
		ImageUrl: "http://example.com/image.jpg",
		Category: m.category,
	}, nil
}

func newConvertTestUseCase(item *entities.BrandItem, category *fashion_dto.CategoryBriefRes) (*BrandItemUseCase, *mockBrandItemRepo) {
	repo := &mockBrandItemRepo{items: map[uuid.UUID]*entities.BrandItem{item.ID: item}}
	return &BrandItemUseCase{
		brandItemRepo:   repo,
		fashionContract: &mockFashionContractWithCategory{category: category},
	}, repo
}

func TestConvertSampleToProduct(t *testing.T) {
	brandID := uuid.New()
	fashionItemID := uuid.New()
	item := &entities.BrandItem{
		AuditableEntity: entities.AuditableEntity{BaseEntity: entities.BaseEntity{ID: uuid.New()}},
		BrandID:         brandID,
		FashionItemID:   &fashionItemID,
		ItemType:        branditemtype.Sample,
		Status:          branditemstatus.Active,
	}
	uc, _ := newConvertTestUseCase(item, &fashion_dto.CategoryBriefRes{ID: uuid.New(), Name: "Dress"})

	res, err := uc.ConvertSampleToProduct(context.Background(), uuid.New(), brandID, item.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ItemType != branditemtype.Product {
		t.Fatalf("expected item type product, got %s", res.ItemType)
	}
}

func TestConvertSampleToProduct_RejectsWrongBrand(t *testing.T) {
	fashionItemID := uuid.New()
	item := &entities.BrandItem{
		AuditableEntity: entities.AuditableEntity{BaseEntity: entities.BaseEntity{ID: uuid.New()}},
		BrandID:         uuid.New(),
		FashionItemID:   &fashionItemID,
		ItemType:        branditemtype.Sample,
		Status:          branditemstatus.Active,
	}
	uc, _ := newConvertTestUseCase(item, &fashion_dto.CategoryBriefRes{ID: uuid.New(), Name: "Dress"})

	_, err := uc.ConvertSampleToProduct(context.Background(), uuid.New(), uuid.New(), item.ID)
	assertStatus(t, err, http.StatusNotFound)
}

func TestConvertSampleToProduct_RejectsAlreadyProduct(t *testing.T) {
	brandID := uuid.New()
	fashionItemID := uuid.New()
	item := &entities.BrandItem{
		AuditableEntity: entities.AuditableEntity{BaseEntity: entities.BaseEntity{ID: uuid.New()}},
		BrandID:         brandID,
		FashionItemID:   &fashionItemID,
		ItemType:        branditemtype.Product,
		Status:          branditemstatus.Active,
	}
	uc, _ := newConvertTestUseCase(item, &fashion_dto.CategoryBriefRes{ID: uuid.New(), Name: "Dress"})

	_, err := uc.ConvertSampleToProduct(context.Background(), uuid.New(), brandID, item.ID)
	assertStatus(t, err, http.StatusConflict)
}

func TestConvertSampleToProduct_RequiresAnalysisCategory(t *testing.T) {
	brandID := uuid.New()
	fashionItemID := uuid.New()
	item := &entities.BrandItem{
		AuditableEntity: entities.AuditableEntity{BaseEntity: entities.BaseEntity{ID: uuid.New()}},
		BrandID:         brandID,
		FashionItemID:   &fashionItemID,
		ItemType:        branditemtype.Sample,
		Status:          branditemstatus.Draft,
	}
	uc, _ := newConvertTestUseCase(item, nil)

	_, err := uc.ConvertSampleToProduct(context.Background(), uuid.New(), brandID, item.ID)
	assertStatus(t, err, http.StatusBadRequest)
}

func TestGetBrandItemFeedbacks_RejectsOtherBrand(t *testing.T) {
	fashionItemID := uuid.New()
	item := &entities.BrandItem{
		AuditableEntity: entities.AuditableEntity{BaseEntity: entities.BaseEntity{ID: uuid.New()}},
		BrandID:         uuid.New(),
		FashionItemID:   &fashionItemID,
		ItemType:        branditemtype.Sample,
		Status:          branditemstatus.Active,
	}
	uc, _ := newConvertTestUseCase(item, nil)
	uc.feedbackRepo = &mockDigitalSampleResponseRepo{feedbacks: map[uuid.UUID]*entities.DigitalSampleResponse{}}

	_, err := uc.GetBrandItemFeedbacks(context.Background(), uuid.New(), uuid.New(), item.ID)
	assertStatus(t, err, http.StatusNotFound)
}

func assertStatus(t *testing.T, err error, status int) {
	t.Helper()
	var appErr *apperror.Error
	if !errors.As(err, &appErr) {
		t.Fatalf("expected *apperror.Error, got %v", err)
	}
	if appErr.Status != status {
		t.Fatalf("expected status %d, got %d", status, appErr.Status)
	}
}
