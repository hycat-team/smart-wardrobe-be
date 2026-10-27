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

// --- recency cooldown preference (033 US3/T054) -----------------------------

func TestApplyRecencyCooldownPromotesEquivalentItemOutsideCooldown(t *testing.T) {
	cooldownTop, cooldownFashion, _ := newRankedTestCandidate("ao", 1.50)
	cooldownTop.Cooldown = true
	outside, outsideFashion, _ := newRankedTestCandidate("ao", 1.40)

	fashionItemMap := map[uuid.UUID]*dto.FashionItemContractDTO{
		cooldownFashion.ID: cooldownFashion,
		outsideFashion.ID:  outsideFashion,
	}

	got, substitutions := ApplyRecencyCooldown([]types.RankedCandidate{cooldownTop, outside}, 0.15, fashionItemMap)

	if substitutions != 1 {
		t.Fatalf("substitutions = %d, want 1", substitutions)
	}
	if got[0].Item.ID != outside.Item.ID {
		t.Fatalf("first candidate = %s, want the item outside cooldown %s", got[0].Item.ID, outside.Item.ID)
	}
	if got[0].Score != 1.40 || got[1].Score != 1.50 {
		t.Fatalf("the base scores must stay untouched, got %.2f and %.2f", got[0].Score, got[1].Score)
	}
}

func TestApplyRecencyCooldownKeepsAllCooldownOrder(t *testing.T) {
	top, topFashion, _ := newRankedTestCandidate("ao", 1.50)
	top.Cooldown = true
	second, secondFashion, _ := newRankedTestCandidate("ao", 1.40)
	second.Cooldown = true
	fashionItemMap := map[uuid.UUID]*dto.FashionItemContractDTO{
		topFashion.ID:    topFashion,
		secondFashion.ID: secondFashion,
	}

	got, substitutions := ApplyRecencyCooldown([]types.RankedCandidate{top, second}, 0.15, fashionItemMap)

	if substitutions != 0 {
		t.Fatalf("substitutions = %d, want 0 when every candidate is in cooldown", substitutions)
	}
	if got[0].Item.ID != top.Item.ID {
		t.Fatalf("first candidate = %s, want the best cooldown item %s", got[0].Item.ID, top.Item.ID)
	}
}

func TestApplyRecencyCooldownIgnoresItemsOutsideTolerance(t *testing.T) {
	cooldownTop, cooldownFashion, _ := newRankedTestCandidate("ao", 1.80)
	cooldownTop.Cooldown = true
	outside, outsideFashion, _ := newRankedTestCandidate("ao", 1.40)
	fashionItemMap := map[uuid.UUID]*dto.FashionItemContractDTO{
		cooldownFashion.ID: cooldownFashion,
		outsideFashion.ID:  outsideFashion,
	}

	got, substitutions := ApplyRecencyCooldown([]types.RankedCandidate{cooldownTop, outside}, 0.15, fashionItemMap)

	if substitutions != 0 {
		t.Fatalf("substitutions = %d, want 0 when the quality gap exceeds the tolerance", substitutions)
	}
	if got[0].Item.ID != cooldownTop.Item.ID {
		t.Fatalf("first candidate = %s, want the better cooldown item %s", got[0].Item.ID, cooldownTop.Item.ID)
	}
}

func TestApplyRecencyCooldownCountsSubstitutionsPerRole(t *testing.T) {
	topCooldown, topFashion, _ := newRankedTestCandidate("ao", 1.50)
	topCooldown.Cooldown = true
	topOutside, topOutsideFashion, _ := newRankedTestCandidate("ao", 1.45)
	bottomCooldown, bottomFashion, _ := newRankedTestCandidate("quan", 1.50)
	bottomCooldown.Cooldown = true
	bottomOutside, bottomOutsideFashion, _ := newRankedTestCandidate("quan", 1.42)
	shoesCooldown, shoesFashion, _ := newRankedTestCandidate("giay", 1.50)
	shoesCooldown.Cooldown = true

	fashionItemMap := map[uuid.UUID]*dto.FashionItemContractDTO{
		topFashion.ID:           topFashion,
		topOutsideFashion.ID:    topOutsideFashion,
		bottomFashion.ID:        bottomFashion,
		bottomOutsideFashion.ID: bottomOutsideFashion,
		shoesFashion.ID:         shoesFashion,
	}

	scored := []types.RankedCandidate{topCooldown, topOutside, bottomCooldown, bottomOutside, shoesCooldown}
	got, substitutions := ApplyRecencyCooldown(scored, 0.15, fashionItemMap)

	if substitutions != 2 {
		t.Fatalf("substitutions = %d, want one per substitutable role (top, bottom)", substitutions)
	}
	promoted := map[uuid.UUID]bool{got[0].Item.ID: true, got[1].Item.ID: true}
	if !promoted[topOutside.Item.ID] || !promoted[bottomOutside.Item.ID] {
		t.Fatalf("expected both promoted items first, got %s then %s", got[0].Item.ID, got[1].Item.ID)
	}
	remaining := map[uuid.UUID]bool{got[2].Item.ID: true, got[3].Item.ID: true, got[4].Item.ID: true}
	if !remaining[topCooldown.Item.ID] || !remaining[bottomCooldown.Item.ID] || !remaining[shoesCooldown.Item.ID] {
		t.Fatalf("cooldown items must stay after the promoted ones, got %v", remaining)
	}
}

func TestApplyRecencyCooldownDoesNotCrossRoles(t *testing.T) {
	cooldownTop, topFashion, _ := newRankedTestCandidate("ao", 1.50)
	cooldownTop.Cooldown = true
	outsideBottom, bottomFashion, _ := newRankedTestCandidate("quan", 1.45)
	fashionItemMap := map[uuid.UUID]*dto.FashionItemContractDTO{
		topFashion.ID:    topFashion,
		bottomFashion.ID: bottomFashion,
	}

	got, substitutions := ApplyRecencyCooldown([]types.RankedCandidate{cooldownTop, outsideBottom}, 0.15, fashionItemMap)

	if substitutions != 0 {
		t.Fatalf("substitutions = %d, want 0: a bottom must never replace a top", substitutions)
	}
	if got[0].Item.ID != cooldownTop.Item.ID {
		t.Fatalf("first candidate = %s, want the top to stay first", got[0].Item.ID)
	}
}
