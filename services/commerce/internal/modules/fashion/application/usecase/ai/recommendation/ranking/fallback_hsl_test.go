package ranking

import (
	"testing"

	"github.com/google/uuid"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/usecase/ai/recommendation/types"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/domain/constants/season"
	wardrobeDto "smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/dto"
)

func TestRunLocalHSLMatchingTreatsChanVayAsBottom(t *testing.T) {
	top, f1 := newHSLTestItem("ao")
	skirt, f2 := newHSLTestItem("chan-vay")
	shoes, f3 := newHSLTestItem("giay")

	fashionItemMap := map[uuid.UUID]*dto.FashionItemContractDTO{
		*top.FashionItemID:   f1,
		*skirt.FashionItemID: f2,
		*shoes.FashionItemID: f3,
	}

	res := RunLocalHSLMatching([]types.CandidateForPrompt{
		{Item: top},
		{Item: skirt},
		{Item: shoes},
	}, dto.RecommendOutfitReq{}, fashionItemMap)

	if len(res.Items) < 2 {
		t.Fatalf("expected at least top and bottom groups, got %+v", res.Items)
	}
	if res.Items[0].Role != "top" || res.Items[1].Role != "bottom" {
		t.Fatalf("unexpected roles returned: %+v", res.Items)
	}
}

func TestRunLocalHSLMatchingTreatsDamAsOnePiece(t *testing.T) {
	dress, f1 := newHSLTestItem("dam")
	shoes, f2 := newHSLTestItem("giay")

	fashionItemMap := map[uuid.UUID]*dto.FashionItemContractDTO{
		*dress.FashionItemID: f1,
		*shoes.FashionItemID: f2,
	}

	res := RunLocalHSLMatching([]types.CandidateForPrompt{
		{Item: dress},
		{Item: shoes},
	}, dto.RecommendOutfitReq{}, fashionItemMap)

	if len(res.Items) == 0 || res.Items[0].Role != "fullbody" {
		t.Fatalf("expected first role fullbody, got %+v", res.Items)
	}
}

func TestRunLocalHSLMatchingRequiresTopAndBottomForSeparates(t *testing.T) {
	pants, f1 := newHSLTestItem("quan")
	shoes, f2 := newHSLTestItem("giay")

	fashionItemMap := map[uuid.UUID]*dto.FashionItemContractDTO{
		*pants.FashionItemID: f1,
		*shoes.FashionItemID: f2,
	}

	res := RunLocalHSLMatching([]types.CandidateForPrompt{
		{Item: pants},
		{Item: shoes},
	}, dto.RecommendOutfitReq{}, fashionItemMap)

	if len(res.Items) != 0 {
		t.Fatalf("expected 0 items when top is missing for separates, got %d", len(res.Items))
	}
}

func newHSLTestItem(slug string) (*wardrobeDto.WardrobeItemContractDTO, *dto.FashionItemContractDTO) {
	fashionID := uuid.New()
	catID := uuid.New()
	name := slug
	item := &wardrobeDto.WardrobeItemContractDTO{
		ID:            uuid.New(),
		FashionItemID: &fashionID,
	}
	res := &dto.FashionItemContractDTO{
		ID: fashionID,
		Category: &dto.CategoryBriefRes{
			ID:   catID,
			Name: name,
			Slug: name,
		},
	}
	return item, res
}

// outfitRoles gom các role có trong outfit thành map để assert thuận tiện.
func outfitRoles(res *dto.RecommendedOutfitRes) map[string]bool {
	roles := make(map[string]bool, len(res.Items))
	for _, group := range res.Items {
		roles[group.Role] = true
	}
	return roles
}

func assertHasRoles(t *testing.T, res *dto.RecommendedOutfitRes, expected ...string) {
	t.Helper()
	roles := outfitRoles(res)
	for _, role := range expected {
		if !roles[role] {
			t.Fatalf("expected role %q in fallback outfit, got roles %+v", role, roles)
		}
	}
}

func TestRunLocalHSLMatchingIncludesHeadwearAndAccessoryForSeparates(t *testing.T) {
	top, f1 := newHSLTestItem("ao")
	pants, f2 := newHSLTestItem("quan")
	shoes, f3 := newHSLTestItem("giay")
	hat, f4 := newHSLTestItem("mu")
	bag, f5 := newHSLTestItem("phu-kien")

	fashionItemMap := map[uuid.UUID]*dto.FashionItemContractDTO{
		*top.FashionItemID:   f1,
		*pants.FashionItemID: f2,
		*shoes.FashionItemID: f3,
		*hat.FashionItemID:   f4,
		*bag.FashionItemID:   f5,
	}

	res := RunLocalHSLMatching([]types.CandidateForPrompt{
		{Item: top}, {Item: pants}, {Item: shoes}, {Item: hat}, {Item: bag},
	}, dto.RecommendOutfitReq{}, fashionItemMap)

	assertHasRoles(t, res, "top", "bottom", "footwear", "headwear", "accessory")
}

func TestRunLocalHSLMatchingIncludesHeadwearAndAccessoryForDress(t *testing.T) {
	dress, f1 := newHSLTestItem("dam")
	shoes, f2 := newHSLTestItem("giay")
	hat, f3 := newHSLTestItem("mu")
	bag, f4 := newHSLTestItem("phu-kien")

	fashionItemMap := map[uuid.UUID]*dto.FashionItemContractDTO{
		*dress.FashionItemID: f1,
		*shoes.FashionItemID: f2,
		*hat.FashionItemID:   f3,
		*bag.FashionItemID:   f4,
	}

	res := RunLocalHSLMatching([]types.CandidateForPrompt{
		{Item: dress}, {Item: shoes}, {Item: hat}, {Item: bag},
	}, dto.RecommendOutfitReq{}, fashionItemMap)

	assertHasRoles(t, res, "fullbody", "footwear", "headwear", "accessory")
}

func TestRunLocalHSLMatchingSkipsAccessoriesWhenCoreOutfitInvalid(t *testing.T) {
	pants, f1 := newHSLTestItem("quan")
	hat, f2 := newHSLTestItem("mu")
	bag, f3 := newHSLTestItem("phu-kien")

	fashionItemMap := map[uuid.UUID]*dto.FashionItemContractDTO{
		*pants.FashionItemID: f1,
		*hat.FashionItemID:   f2,
		*bag.FashionItemID:   f3,
	}

	res := RunLocalHSLMatching([]types.CandidateForPrompt{
		{Item: pants}, {Item: hat}, {Item: bag},
	}, dto.RecommendOutfitReq{}, fashionItemMap)

	if len(res.Items) != 0 {
		t.Fatalf("expected 0 items when core outfit is invalid, got %d", len(res.Items))
	}
}

func TestRunLocalHSLMatchingIncludesOuterwearForWinterSeasonWithoutWeather(t *testing.T) {
	top, f1 := newHSLTestItem("ao")
	pants, f2 := newHSLTestItem("quan")
	outer, f3 := newHSLTestItem("ao-khoac")

	fashionItemMap := map[uuid.UUID]*dto.FashionItemContractDTO{
		*top.FashionItemID:   f1,
		*pants.FashionItemID: f2,
		*outer.FashionItemID: f3,
	}

	winter := season.Winter
	res := RunLocalHSLMatching([]types.CandidateForPrompt{
		{Item: top}, {Item: pants}, {Item: outer},
	}, dto.RecommendOutfitReq{Season: &winter}, fashionItemMap)

	assertHasRoles(t, res, "top", "bottom", "outerwear")
}

func TestRunLocalHSLMatchingIncludesOuterwearForAutumnSeason(t *testing.T) {
	top, f1 := newHSLTestItem("ao")
	pants, f2 := newHSLTestItem("quan")
	outer, f3 := newHSLTestItem("ao-khoac")

	fashionItemMap := map[uuid.UUID]*dto.FashionItemContractDTO{
		*top.FashionItemID:   f1,
		*pants.FashionItemID: f2,
		*outer.FashionItemID: f3,
	}

	autumn := season.Autumn
	res := RunLocalHSLMatching([]types.CandidateForPrompt{
		{Item: top}, {Item: pants}, {Item: outer},
	}, dto.RecommendOutfitReq{Season: &autumn}, fashionItemMap)

	assertHasRoles(t, res, "outerwear")
}

func TestRunLocalHSLMatchingExcludesOuterwearForHotWeather(t *testing.T) {
	top, f1 := newHSLTestItem("ao")
	pants, f2 := newHSLTestItem("quan")
	outer, f3 := newHSLTestItem("ao-khoac")

	fashionItemMap := map[uuid.UUID]*dto.FashionItemContractDTO{
		*top.FashionItemID:   f1,
		*pants.FashionItemID: f2,
		*outer.FashionItemID: f3,
	}

	weather := "nắng"
	res := RunLocalHSLMatching([]types.CandidateForPrompt{
		{Item: top}, {Item: pants}, {Item: outer},
	}, dto.RecommendOutfitReq{Weather: &weather}, fashionItemMap)

	assertHasRoles(t, res, "top", "bottom")
	if outfitRoles(res)["outerwear"] {
		t.Fatalf("expected no outerwear in hot weather, got %+v", res.Items)
	}
}

func TestShouldIncludeOuterwear(t *testing.T) {
	tests := []struct {
		name  string
		input dto.RecommendOutfitReq
		want  bool
	}{
		{
			name:  "no context at all",
			input: dto.RecommendOutfitReq{},
			want:  false,
		},
		{
			name:  "cold weather",
			input: dto.RecommendOutfitReq{Weather: ptr("trời lạnh")},
			want:  true,
		},
		{
			name:  "rainy weather",
			input: dto.RecommendOutfitReq{Weather: ptr("mưa rào")},
			want:  true,
		},
		{
			name:  "hot weather overrides season",
			input: dto.RecommendOutfitReq{Weather: ptr("nắng"), Season: seasonPtr(season.Winter)},
			want:  false,
		},
		{
			name:  "winter season",
			input: dto.RecommendOutfitReq{Season: seasonPtr(season.Winter)},
			want:  true,
		},
		{
			name:  "summer season",
			input: dto.RecommendOutfitReq{Season: seasonPtr(season.Summer)},
			want:  false,
		},
		{
			name:  "layering intent in details",
			input: dto.RecommendOutfitReq{Details: ptr("phối đồ nhiều lớp, cần áo khoác")},
			want:  true,
		},
		{
			name:  "plain details without layering",
			input: dto.RecommendOutfitReq{Details: ptr("đi làm văn phòng")},
			want:  false,
		},
		{
			name:  "all season without signal",
			input: dto.RecommendOutfitReq{Season: seasonPtr(season.All)},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldIncludeOuterwear(tt.input); got != tt.want {
				t.Fatalf("shouldIncludeOuterwear() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRunLocalHSLMatchingNeverEmitsOtherRole(t *testing.T) {
	top, f1 := newHSLTestItem("ao")
	pants, f2 := newHSLTestItem("quan")
	unclassified, f3 := newHSLTestItem("other")

	fashionItemMap := map[uuid.UUID]*dto.FashionItemContractDTO{
		*top.FashionItemID:          f1,
		*pants.FashionItemID:        f2,
		*unclassified.FashionItemID: f3,
	}

	res := RunLocalHSLMatching([]types.CandidateForPrompt{
		{Item: top}, {Item: pants}, {Item: unclassified},
	}, dto.RecommendOutfitReq{}, fashionItemMap)

	for _, group := range res.Items {
		if group.Role != "top" && group.Role != "bottom" && group.Role != "footwear" &&
			group.Role != "fullbody" && group.Role != "outerwear" &&
			group.Role != "headwear" && group.Role != "accessory" {
			t.Fatalf("unexpected role %q outside the allowed wearing role enum", group.Role)
		}
	}
	if outfitRoles(res)["accessory"] != true {
		t.Fatalf("expected unclassified item to be grouped as accessory, got %+v", res.Items)
	}
}

func ptr(value string) *string {
	return &value
}

func seasonPtr(value season.Season) *season.Season {
	return &value
}
