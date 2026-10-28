package brand

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"smart-wardrobe-be/services/commerce/internal/modules/brand/application/dto"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/constants/brandstatus"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/entities"
	"smart-wardrobe-be/shared/application/constants/apperror"

	"github.com/google/uuid"
)

func assertAppErrorStatus(t *testing.T, err error, status int) {
	t.Helper()
	var appErr *apperror.Error
	if !errors.As(err, &appErr) {
		t.Fatalf("expected *apperror.Error, got %v", err)
	}
	if appErr.Status != status {
		t.Fatalf("expected status %d, got %d (%s)", status, appErr.Status, appErr.Message)
	}
}

func newBrand(id uuid.UUID, slug string, status brandstatus.BrandStatus) *entities.Brand {
	return &entities.Brand{Slug: slug, Name: slug, Status: status}
}

func TestNormalizeBrandSlug(t *testing.T) {
	cases := map[string]string{
		"  Mori Studio ": "mori-studio",
		"MORI":           "mori",
		"a--b":           "a-b",
		"mori   studio":  "mori-studio",
	}

	for input, want := range cases {
		if got := NormalizeBrandSlug(input); got != want {
			t.Errorf("NormalizeBrandSlug(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestResolveBrandRef(t *testing.T) {
	brandID := uuid.New()
	brand := newBrand(brandID, "mori-studio", brandstatus.Active)
	brand.ID = brandID
	repo := &mockBrandRepo{brands: map[uuid.UUID]*entities.Brand{brandID: brand}}
	uc := &BrandUseCase{brandRepo: repo}

	t.Run("resolves by UUID", func(t *testing.T) {
		got, err := uc.ResolveBrandRef(context.Background(), brandID.String())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != brandID {
			t.Fatalf("got %s, want %s", got, brandID)
		}
	})

	t.Run("resolves by slug case-insensitively", func(t *testing.T) {
		got, err := uc.ResolveBrandRef(context.Background(), "  MORI-STUDIO ")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != brandID {
			t.Fatalf("got %s, want %s", got, brandID)
		}
	})

	t.Run("unknown slug returns not found", func(t *testing.T) {
		_, err := uc.ResolveBrandRef(context.Background(), "missing-brand")
		assertAppErrorStatus(t, err, http.StatusNotFound)
	})

	t.Run("malformed slug returns not found", func(t *testing.T) {
		_, err := uc.ResolveBrandRef(context.Background(), "bad_slug")
		assertAppErrorStatus(t, err, http.StatusNotFound)
	})
}

func TestResolveActiveBrandRefRejectsInactive(t *testing.T) {
	brandID := uuid.New()
	brand := newBrand(brandID, "mori-studio", brandstatus.PendingReview)
	brand.ID = brandID
	repo := &mockBrandRepo{brands: map[uuid.UUID]*entities.Brand{brandID: brand}}
	uc := &BrandUseCase{brandRepo: repo}

	_, err := uc.ResolveActiveBrandRef(context.Background(), "mori-studio")
	assertAppErrorStatus(t, err, http.StatusNotFound)
}

func TestCreateBrandRequestValidatesSlug(t *testing.T) {
	repo := &mockBrandRepo{brands: map[uuid.UUID]*entities.Brand{}}
	uc := &BrandUseCase{brandRepo: repo}

	_, err := uc.CreateBrandRequest(context.Background(), uuid.New(), dto.CreateBrandReq{
		Name: "Bad Brand",
		Slug: "admin",
	})
	assertAppErrorStatus(t, err, http.StatusBadRequest)
}

func TestCreateBrandRequestRejectsDuplicateSlug(t *testing.T) {
	existingID := uuid.New()
	existing := newBrand(existingID, "mori-studio", brandstatus.Active)
	existing.ID = existingID
	repo := &mockBrandRepo{brands: map[uuid.UUID]*entities.Brand{existingID: existing}}
	uc := &BrandUseCase{brandRepo: repo}

	_, err := uc.CreateBrandRequest(context.Background(), uuid.New(), dto.CreateBrandReq{
		Name: "Mori Studio 2",
		Slug: "mori-studio",
	})
	assertAppErrorStatus(t, err, http.StatusConflict)
}
