package mapper

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/usecase/ai/recommendation/types"
	wardrobeDto "smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/dto"
)

// newTestCandidate tạo một ứng viên kèm fashion item với category slug cho trước.
func newTestCandidate(t *testing.T, slug string) (types.CandidateForPrompt, *dto.FashionItemContractDTO) {
	t.Helper()
	fashionID := uuid.New()
	fashion := &dto.FashionItemContractDTO{
		ID: fashionID,
		Category: &dto.CategoryBriefRes{
			ID:   uuid.New(),
			Name: slug,
			Slug: slug,
		},
	}
	item := &wardrobeDto.WardrobeItemContractDTO{
		ID:            uuid.New(),
		FashionItemID: &fashionID,
	}
	return types.CandidateForPrompt{Item: item}, fashion
}

// parseLlmFixture dựng phản hồi LLM từ JSON để test gần với dữ liệu thật.
func parseLlmFixture(t *testing.T, raw string) types.LlmOutfitResponse {
	t.Helper()
	var res types.LlmOutfitResponse
	if err := json.Unmarshal([]byte(raw), &res); err != nil {
		t.Fatalf("failed to parse llm fixture: %v", err)
	}
	return res
}

func countRole(groups []*dto.RecommendedItemGroup, role string) int {
	count := 0
	for _, group := range groups {
		if group.Role == role {
			count++
		}
	}
	return count
}

func rolesOf(groups []*dto.RecommendedItemGroup) []string {
	roles := make([]string, 0, len(groups))
	for _, group := range groups {
		roles = append(roles, group.Role)
	}
	return roles
}

func TestMapLLMResponseToGroupsBackfillsMissingAccessory(t *testing.T) {
	top, fTop := newTestCandidate(t, "ao")
	pants, fPants := newTestCandidate(t, "quan")
	shoes, fShoes := newTestCandidate(t, "giay")
	hat, fHat := newTestCandidate(t, "mu")
	bag, fBag := newTestCandidate(t, "phu-kien")

	candidates := []types.CandidateForPrompt{top, pants, shoes, hat, bag}
	fashionItemMap := map[uuid.UUID]*dto.FashionItemContractDTO{
		fTop.ID: fTop, fPants.ID: fPants, fShoes.ID: fShoes, fHat.ID: fHat, fBag.ID: fBag,
	}

	// LLM chỉ trả về mũ, bỏ sót phụ kiện.
	llmRes := parseLlmFixture(t, `{"title":"t","explanation":"e","items":[
		{"role":"top","primary_id":"`+top.Item.ID.String()+`","alternative_ids":[]},
		{"role":"bottom","primary_id":"`+pants.Item.ID.String()+`","alternative_ids":[]},
		{"role":"footwear","primary_id":"`+shoes.Item.ID.String()+`","alternative_ids":[]},
		{"role":"headwear","primary_id":"`+hat.Item.ID.String()+`","alternative_ids":[]}
	]}`)

	groups := MapLLMResponseToGroups(candidates, llmRes, fashionItemMap)

	if countRole(groups, "headwear") != 1 {
		t.Fatalf("expected exactly 1 headwear group, got roles %+v", rolesOf(groups))
	}
	if countRole(groups, "accessory") != 1 {
		t.Fatalf("expected accessory to be backfilled, got roles %+v", rolesOf(groups))
	}
}

func TestMapLLMResponseToGroupsBackfillsBothLayerRolesWhenLlmOmitsBoth(t *testing.T) {
	top, fTop := newTestCandidate(t, "ao")
	pants, fPants := newTestCandidate(t, "quan")
	hat, fHat := newTestCandidate(t, "mu")
	bag, fBag := newTestCandidate(t, "phu-kien")

	candidates := []types.CandidateForPrompt{top, pants, hat, bag}
	fashionItemMap := map[uuid.UUID]*dto.FashionItemContractDTO{
		fTop.ID: fTop, fPants.ID: fPants, fHat.ID: fHat, fBag.ID: fBag,
	}

	llmRes := parseLlmFixture(t, `{"title":"t","explanation":"e","items":[
		{"role":"top","primary_id":"`+top.Item.ID.String()+`","alternative_ids":[]},
		{"role":"bottom","primary_id":"`+pants.Item.ID.String()+`","alternative_ids":[]}
	]}`)

	groups := MapLLMResponseToGroups(candidates, llmRes, fashionItemMap)

	if countRole(groups, "headwear") != 1 || countRole(groups, "accessory") != 1 {
		t.Fatalf("expected both layer roles backfilled, got roles %+v", rolesOf(groups))
	}
}

func TestMapLLMResponseToGroupsDoesNotDuplicateLayerRolesFromLlm(t *testing.T) {
	top, fTop := newTestCandidate(t, "ao")
	pants, fPants := newTestCandidate(t, "quan")
	hat, fHat := newTestCandidate(t, "mu")
	bag, fBag := newTestCandidate(t, "phu-kien")
	otherHat, fOtherHat := newTestCandidate(t, "mu")

	candidates := []types.CandidateForPrompt{top, pants, hat, bag, otherHat}
	fashionItemMap := map[uuid.UUID]*dto.FashionItemContractDTO{
		fTop.ID: fTop, fPants.ID: fPants, fHat.ID: fHat, fBag.ID: fBag, fOtherHat.ID: fOtherHat,
	}

	llmRes := parseLlmFixture(t, `{"title":"t","explanation":"e","items":[
		{"role":"top","primary_id":"`+top.Item.ID.String()+`","alternative_ids":[]},
		{"role":"bottom","primary_id":"`+pants.Item.ID.String()+`","alternative_ids":[]},
		{"role":"headwear","primary_id":"`+hat.Item.ID.String()+`","alternative_ids":[]},
		{"role":"accessory","primary_id":"`+bag.Item.ID.String()+`","alternative_ids":[]}
	]}`)

	groups := MapLLMResponseToGroups(candidates, llmRes, fashionItemMap)

	if got := countRole(groups, "headwear"); got != 1 {
		t.Fatalf("expected exactly 1 headwear group, got %d (roles %+v)", got, rolesOf(groups))
	}
	if got := countRole(groups, "accessory"); got != 1 {
		t.Fatalf("expected exactly 1 accessory group, got %d (roles %+v)", got, rolesOf(groups))
	}
	if got := len(groups); got != 4 {
		t.Fatalf("expected 4 groups, got %d (roles %+v)", got, rolesOf(groups))
	}
}

func TestMapLLMResponseToGroupsKeepsLayerRolesAndDropsTopBottomWithFullbody(t *testing.T) {
	dress, fDress := newTestCandidate(t, "dam")
	shoes, fShoes := newTestCandidate(t, "giay")
	top, fTop := newTestCandidate(t, "ao")
	hat, fHat := newTestCandidate(t, "mu")
	bag, fBag := newTestCandidate(t, "phu-kien")

	candidates := []types.CandidateForPrompt{dress, shoes, top, hat, bag}
	fashionItemMap := map[uuid.UUID]*dto.FashionItemContractDTO{
		fDress.ID: fDress, fShoes.ID: fShoes, fTop.ID: fTop, fHat.ID: fHat, fBag.ID: fBag,
	}

	// LLM trả cả đầm lẫn áo (vi phạm) kèm mũ và phụ kiện.
	llmRes := parseLlmFixture(t, `{"title":"t","explanation":"e","items":[
		{"role":"fullbody","primary_id":"`+dress.Item.ID.String()+`","alternative_ids":[]},
		{"role":"footwear","primary_id":"`+shoes.Item.ID.String()+`","alternative_ids":[]},
		{"role":"top","primary_id":"`+top.Item.ID.String()+`","alternative_ids":[]},
		{"role":"headwear","primary_id":"`+hat.Item.ID.String()+`","alternative_ids":[]},
		{"role":"accessory","primary_id":"`+bag.Item.ID.String()+`","alternative_ids":[]}
	]}`)

	groups := MapLLMResponseToGroups(candidates, llmRes, fashionItemMap)

	if countRole(groups, "top") != 0 {
		t.Fatalf("expected top to be dropped with fullbody, got roles %+v", rolesOf(groups))
	}
	if countRole(groups, "bottom") != 0 {
		t.Fatalf("expected bottom to stay absent with fullbody, got roles %+v", rolesOf(groups))
	}
	if countRole(groups, "fullbody") != 1 {
		t.Fatalf("expected fullbody kept, got roles %+v", rolesOf(groups))
	}
	if countRole(groups, "headwear") != 1 || countRole(groups, "accessory") != 1 {
		t.Fatalf("expected layer roles kept with fullbody, got roles %+v", rolesOf(groups))
	}
}

func TestMapLLMResponseToGroupsSkipsRoleWhenPoolHasNoCandidate(t *testing.T) {
	top, fTop := newTestCandidate(t, "ao")
	pants, fPants := newTestCandidate(t, "quan")
	hat, fHat := newTestCandidate(t, "mu")

	candidates := []types.CandidateForPrompt{top, pants, hat}
	fashionItemMap := map[uuid.UUID]*dto.FashionItemContractDTO{
		fTop.ID: fTop, fPants.ID: fPants, fHat.ID: fHat,
	}

	llmRes := parseLlmFixture(t, `{"title":"t","explanation":"e","items":[
		{"role":"top","primary_id":"`+top.Item.ID.String()+`","alternative_ids":[]},
		{"role":"bottom","primary_id":"`+pants.Item.ID.String()+`","alternative_ids":[]}
	]}`)

	groups := MapLLMResponseToGroups(candidates, llmRes, fashionItemMap)

	if countRole(groups, "accessory") != 0 {
		t.Fatalf("expected no accessory group when pool has none, got roles %+v", rolesOf(groups))
	}
	if countRole(groups, "headwear") != 1 {
		t.Fatalf("expected headwear backfilled from pool, got roles %+v", rolesOf(groups))
	}
	for _, group := range groups {
		if group.Primary == nil {
			t.Fatalf("group %q must not be emitted with nil primary", group.Role)
		}
	}
}

func TestMapLLMResponseToGroupsBackfillsMissingCoreSeparates(t *testing.T) {
	top, fTop := newTestCandidate(t, "ao")
	pants, fPants := newTestCandidate(t, "quan")

	candidates := []types.CandidateForPrompt{top, pants}
	fashionItemMap := map[uuid.UUID]*dto.FashionItemContractDTO{
		fTop.ID: fTop, fPants.ID: fPants,
	}

	llmRes := parseLlmFixture(t, `{"title":"t","explanation":"e","items":[
		{"role":"top","primary_id":"`+top.Item.ID.String()+`","alternative_ids":[]}
	]}`)

	groups := MapLLMResponseToGroups(candidates, llmRes, fashionItemMap)

	if countRole(groups, "top") != 1 || countRole(groups, "bottom") != 1 {
		t.Fatalf("expected core separates backfilled, got roles %+v", rolesOf(groups))
	}
}

func TestCategorySlugToWearingRoleTrimsWhitespace(t *testing.T) {
	tests := []struct {
		slug string
		want string
	}{
		{slug: " mu ", want: "headwear"},
		{slug: "PHU-KIEN", want: "accessory"},
		{slug: "  ao-khoac\t", want: "outerwear"},
		{slug: "quan", want: "bottom"},
		{slug: "khong-ro", want: "other"},
	}

	for _, tt := range tests {
		if got := CategorySlugToWearingRole(tt.slug); got != tt.want {
			t.Fatalf("CategorySlugToWearingRole(%q) = %q, want %q", tt.slug, got, tt.want)
		}
	}
}
