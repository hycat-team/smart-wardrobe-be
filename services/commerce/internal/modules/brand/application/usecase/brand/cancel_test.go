package brand

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/constants/brandmemberstatus"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/constants/brandstatus"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/entities"
	"smart-wardrobe-be/shared/application/constants/apperror"
	"smart-wardrobe-be/shared/domain/constants/brandmemberrole"

	"github.com/google/uuid"
)

func cancelTestUseCase(brand *entities.Brand) (*BrandUseCase, *mockBrandRepo) {
	brandRepo := &mockBrandRepo{brands: map[uuid.UUID]*entities.Brand{brand.ID: brand}}
	member := &entities.BrandMember{
		BrandID: brand.ID,
		UserID:  brand.CreatedByUserID,
		Role:    brandmemberrole.Owner,
		Status:  brandmemberstatus.Active,
	}
	member.ID = uuid.New()
	memberRepo := &memberTestMemberRepo{members: map[uuid.UUID]*entities.BrandMember{member.ID: member}}
	return &BrandUseCase{brandRepo: brandRepo, memberRepo: memberRepo}, brandRepo
}

func newCancelTestBrand(status brandstatus.BrandStatus) *entities.Brand {
	brand := &entities.Brand{
		Status:          status,
		CreatedByUserID: uuid.New(),
		Slug:            "mori-studio",
		Name:            "Mori Studio",
	}
	brand.ID = uuid.New()
	return brand
}

func TestCancelBrandRequest_Pending(t *testing.T) {
	brand := newCancelTestBrand(brandstatus.PendingReview)
	uc, repo := cancelTestUseCase(brand)

	if err := uc.CancelBrandRequest(context.Background(), brand.CreatedByUserID, brand.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, exists := repo.brands[brand.ID]; exists {
		t.Fatal("expected brand request to be deleted")
	}
}

func TestCancelBrandRequest_Rejected(t *testing.T) {
	brand := newCancelTestBrand(brandstatus.Rejected)
	uc, repo := cancelTestUseCase(brand)

	if err := uc.CancelBrandRequest(context.Background(), brand.CreatedByUserID, brand.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, exists := repo.brands[brand.ID]; exists {
		t.Fatal("expected rejected brand to be deleted")
	}
}

func TestCancelBrandRequest_RejectsActive(t *testing.T) {
	brand := newCancelTestBrand(brandstatus.Active)
	uc, _ := cancelTestUseCase(brand)

	err := uc.CancelBrandRequest(context.Background(), brand.CreatedByUserID, brand.ID)
	assertCancelStatus(t, err, http.StatusConflict)
}

func TestCancelBrandRequest_RejectsOtherUser(t *testing.T) {
	brand := newCancelTestBrand(brandstatus.PendingReview)
	uc, _ := cancelTestUseCase(brand)

	err := uc.CancelBrandRequest(context.Background(), uuid.New(), brand.ID)
	assertCancelStatus(t, err, http.StatusForbidden)
}

func TestCancelBrandRequest_UnknownBrand(t *testing.T) {
	brand := newCancelTestBrand(brandstatus.PendingReview)
	uc, _ := cancelTestUseCase(brand)

	err := uc.CancelBrandRequest(context.Background(), brand.CreatedByUserID, uuid.New())
	assertCancelStatus(t, err, http.StatusNotFound)
}

func assertCancelStatus(t *testing.T, err error, status int) {
	t.Helper()
	var appErr *apperror.Error
	if !errors.As(err, &appErr) {
		t.Fatalf("expected *apperror.Error, got %v", err)
	}
	if appErr.Status != status {
		t.Fatalf("expected status %d, got %d", status, appErr.Status)
	}
}
