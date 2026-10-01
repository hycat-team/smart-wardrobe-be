package ranking

import (
	"testing"

	"github.com/google/uuid"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/usecase/ai/recommendation/types"
	wardrobeDto "smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/dto"
)

// newRankedTestCandidate tạo một ứng viên đã xếp hạng với category slug cho trước.
func newRankedTestCandidate(slug string, score float64) (types.RankedCandidate, *dto.FashionItemContractDTO, *wardrobeDto.WardrobeItemContractDTO) {
	fashionID := uuid.New()
	catID := uuid.New()
	item := &wardrobeDto.WardrobeItemContractDTO{
		ID:            uuid.New(),
		FashionItemID: &fashionID,
	}
	fashion := &dto.FashionItemContractDTO{
		ID: fashionID,
		Category: &dto.CategoryBriefRes{
			ID:   catID,
			Name: slug,
			Slug: slug,
		},
	}
	candidate := types.RankedCandidate{
		Item:  item,
		Score: score,
		Tags:  []string{"candidate-source:" + string(types.CandidateSourceRetrieval)},
	}
	return candidate, fashion, item
}

func rolesOfRanked(candidates []types.RankedCandidate, fashionItemMap map[uuid.UUID]*dto.FashionItemContractDTO) map[string]bool {
	roles := map[string]bool{}
	for _, candidate := range candidates {
		roles[rankedWearingRole(candidate, fashionItemMap)] = true
	}
	return roles
}

func TestEnsureRankedOutfitStructureBackfillsOptionalLayeringRoles(t *testing.T) {
	top, fTop, _ := newRankedTestCandidate("ao", 1.5)
	pants, fPants, _ := newRankedTestCandidate("quan", 1.4)
	hat, fHat, _ := newRankedTestCandidate("mu", 0.8)
	bag, fBag, _ := newRankedTestCandidate("phu-kien", 0.7)
	outer, fOuter, _ := newRankedTestCandidate("ao-khoac", 0.6)

	fashionItemMap := map[uuid.UUID]*dto.FashionItemContractDTO{
		fTop.ID:   fTop,
		fPants.ID: fPants,
		fHat.ID:   fHat,
		fBag.ID:   fBag,
		fOuter.ID: fOuter,
	}

	// Pool đã diversify chỉ giữ lại top + bottom: đa dạng hóa theo trần danh mục
	// đã loại mũ, phụ kiện và áo khoác dù chúng vẫn còn trong pool đã chấm điểm.
	selected := []types.RankedCandidate{top, pants}
	all := []types.RankedCandidate{top, pants, hat, bag, outer}

	got := ensureRankedOutfitStructure(selected, all, fashionItemMap)

	roles := rolesOfRanked(got, fashionItemMap)
	for _, role := range []string{"top", "bottom", "headwear", "accessory", "outerwear"} {
		if !roles[role] {
			t.Fatalf("expected role %q after backfill, got %+v", role, roles)
		}
	}
}

func TestEnsureRankedOutfitStructureKeepsExistingOptionalRoles(t *testing.T) {
	top, fTop, _ := newRankedTestCandidate("ao", 1.5)
	pants, fPants, _ := newRankedTestCandidate("quan", 1.4)
	hat, fHat, _ := newRankedTestCandidate("mu", 1.6)
	bag, fBag, _ := newRankedTestCandidate("phu-kien", 1.3)
	otherHat, fOtherHat, _ := newRankedTestCandidate("mu", 1.2)

	fashionItemMap := map[uuid.UUID]*dto.FashionItemContractDTO{
		fTop.ID:      fTop,
		fPants.ID:    fPants,
		fHat.ID:      fHat,
		fBag.ID:      fBag,
		fOtherHat.ID: fOtherHat,
	}

	selected := []types.RankedCandidate{top, pants, hat, bag}
	all := append([]types.RankedCandidate{}, selected...)
	all = append(all, otherHat)

	got := ensureRankedOutfitStructure(selected, all, fashionItemMap)

	if len(got) != 4 {
		t.Fatalf("expected pool size unchanged when optional roles exist, got %d", len(got))
	}
}

func TestEnsureRankedOutfitStructureBackfillsCoreRoles(t *testing.T) {
	top, fTop, _ := newRankedTestCandidate("ao", 1.5)
	pants, fPants, _ := newRankedTestCandidate("quan", 1.4)

	fashionItemMap := map[uuid.UUID]*dto.FashionItemContractDTO{
		fTop.ID:   fTop,
		fPants.ID: fPants,
	}

	selected := []types.RankedCandidate{top}
	all := []types.RankedCandidate{top, pants}

	got := ensureRankedOutfitStructure(selected, all, fashionItemMap)

	roles := rolesOfRanked(got, fashionItemMap)
	if !roles["top"] || !roles["bottom"] {
		t.Fatalf("expected core roles backfilled, got %+v", roles)
	}
}

func TestMissingOptionalRolesReturnsNilWhenAllPresent(t *testing.T) {
	hat, fHat, _ := newRankedTestCandidate("mu", 1.0)
	bag, fBag, _ := newRankedTestCandidate("phu-kien", 1.0)
	outer, fOuter, _ := newRankedTestCandidate("ao-khoac", 1.0)

	fashionItemMap := map[uuid.UUID]*dto.FashionItemContractDTO{
		fHat.ID:   fHat,
		fBag.ID:   fBag,
		fOuter.ID: fOuter,
	}

	candidates := []types.RankedCandidate{hat, bag, outer}
	if need := missingOptionalRoles(candidates, fashionItemMap); need != nil {
		t.Fatalf("expected nil when all optional roles present, got %+v", need)
	}

	if need := missingOptionalRoles([]types.RankedCandidate{hat}, fashionItemMap); len(need) != 2 {
		t.Fatalf("expected 2 missing optional roles, got %+v", need)
	}
}
