package ranking

import (
	"testing"

	"github.com/google/uuid"
	"smart-wardrobe-be/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/internal/modules/fashion/application/usecase/ai/recommendation/types"
	wardrobeDto "smart-wardrobe-be/internal/modules/wardrobe/application/dto"
)

func TestRunLocalHSLMatchingTreatsChanVayAsBottom(t *testing.T) {
	top, f1 := newHSLTestItem("ao")
	skirt, f2 := newHSLTestItem("chan-vay")
	shoes, f3 := newHSLTestItem("giay")

	fashionItemMap := map[uuid.UUID]*dto.FashionItemRes{
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

	fashionItemMap := map[uuid.UUID]*dto.FashionItemRes{
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

	fashionItemMap := map[uuid.UUID]*dto.FashionItemRes{
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

func newHSLTestItem(slug string) (*wardrobeDto.WardrobeItemContractDTO, *dto.FashionItemRes) {
	fashionID := uuid.New()
	catID := uuid.New()
	name := slug
	item := &wardrobeDto.WardrobeItemContractDTO{
		ID:            uuid.New(),
		FashionItemID: &fashionID,
	}
	res := &dto.FashionItemRes{
		ID: fashionID,
		Category: &dto.CategoryBriefRes{
			ID:   catID,
			Name: name,
			Slug: name,
		},
	}
	return item, res
}
