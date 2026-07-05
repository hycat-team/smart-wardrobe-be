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
		top.FashionItem.ID:   f1,
		skirt.FashionItem.ID: f2,
		shoes.FashionItem.ID: f3,
	}

	res := RunLocalHSLMatching([]types.CandidateForPrompt{
		{Item: top},
		{Item: skirt},
		{Item: shoes},
	}, dto.RecommendOutfitReq{}, fashionItemMap)

	if len(res.Items) < 2 {
		t.Fatalf("expected at least top and bottom groups, got %+v", res.Items)
	}
	if res.Items[0].Role != "ao" || res.Items[1].Role != "quan" && res.Items[1].Role != "chan-vay" {
		t.Fatalf("unexpected roles returned: %+v", res.Items)
	}
}

func TestRunLocalHSLMatchingTreatsDamAsOnePiece(t *testing.T) {
	dress, f1 := newHSLTestItem("dam")
	shoes, f2 := newHSLTestItem("giay")

	fashionItemMap := map[uuid.UUID]*dto.FashionItemRes{
		dress.FashionItem.ID: f1,
		shoes.FashionItem.ID: f2,
	}

	res := RunLocalHSLMatching([]types.CandidateForPrompt{
		{Item: dress},
		{Item: shoes},
	}, dto.RecommendOutfitReq{}, fashionItemMap)

	if len(res.Items) == 0 || res.Items[0].Role != "dam" {
		t.Fatalf("expected first role dam, got %+v", res.Items)
	}
}

func newHSLTestItem(slug string) (*wardrobeDto.WardrobeItemRes, *dto.FashionItemRes) {
	fashionID := uuid.New()
	catID := uuid.New()
	name := slug
	item := &wardrobeDto.WardrobeItemRes{
		ID: uuid.New(),
		FashionItem: &wardrobeDto.FashionItemRes{
			ID: fashionID,
		},
	}
	res := &dto.FashionItemRes{
		ID:           fashionID,
		CategoryID:   &catID,
		CategoryName: &name,
	}
	return item, res
}
