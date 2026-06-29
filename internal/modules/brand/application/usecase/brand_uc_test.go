package usecase

import (
	"context"
	"testing"

	"smart-wardrobe-be/internal/modules/brand/application/dto"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandstatus"
	"smart-wardrobe-be/internal/shared/domain/entities"

	"github.com/google/uuid"
)

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
		brandRepo: brandRepo,
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
