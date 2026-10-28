package benefit

import (
	"context"
	"testing"

	"smart-wardrobe-be/services/commerce/internal/modules/brand/application/dto"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/constants/benefit/benefitstatus"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/constants/benefit/benefittype"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/constants/benefit/benefitunlocktype"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/entities"

	"github.com/google/uuid"
)

func TestListBrandBenefitsForStaff_FiltersStatusAndTier(t *testing.T) {
	brandID := uuid.New()
	tierID := uuid.New()

	archived := &entities.BrandBenefit{
		BrandID:        brandID,
		Name:           "Archived perk",
		BenefitType:    benefittype.Voucher,
		UnlockType:     benefitunlocktype.TierPrivilege,
		Status:         benefitstatus.Archived,
		RequiredTierID: &tierID,
	}
	active := &entities.BrandBenefit{
		BrandID:     brandID,
		Name:        "Active voucher",
		BenefitType: benefittype.Voucher,
		UnlockType:  benefitunlocktype.PointRedemption,
		Status:      benefitstatus.Active,
	}

	repo := &mockBenefitRepo{benefits: map[uuid.UUID]*entities.BrandBenefit{
		uuid.New(): archived,
		uuid.New(): active,
	}}
	uc := &BrandBenefitUseCase{benefitRepo: repo}

	status := "archived"
	tier := tierID.String()
	res, err := uc.ListBrandBenefitsForStaff(context.Background(), uuid.New(), brandID, dto.GetBrandBenefitsQueryReq{
		Status:         &status,
		RequiredTierID: &tier,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Items) != 1 {
		t.Fatalf("expected 1 benefit, got %d", len(res.Items))
	}
	if res.Items[0].Name != "Archived perk" {
		t.Fatalf("unexpected benefit: %s", res.Items[0].Name)
	}

	all, err := uc.ListBrandBenefitsForStaff(context.Background(), uuid.New(), brandID, dto.GetBrandBenefitsQueryReq{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(all.Items) != 2 {
		t.Fatalf("expected staff list to include every status, got %d", len(all.Items))
	}
}
