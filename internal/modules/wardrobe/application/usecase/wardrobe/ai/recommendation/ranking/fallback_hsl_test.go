package ranking

import (
	"testing"

	"github.com/google/uuid"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/application/usecase/wardrobe/ai/recommendation/types"
	"smart-wardrobe-be/internal/shared/domain/entities"
)

func TestRunLocalHSLMatchingTreatsChanVayAsBottom(t *testing.T) {
	top := wardrobeItem("ao")
	skirt := wardrobeItem("chan-vay")
	shoes := wardrobeItem("giay")

	res := RunLocalHSLMatching([]types.CandidateForPrompt{
		{Item: top},
		{Item: skirt},
		{Item: shoes},
	}, dto.RecommendOutfitReq{})

	if len(res.Items) < 2 {
		t.Fatalf("expected at least top and bottom groups, got %+v", res.Items)
	}
	if res.Items[0].Role != "ao" || res.Items[1].Role != "quan" && res.Items[1].Role != "chan-vay" {
		t.Fatalf("unexpected roles returned: %+v", res.Items)
	}
}

func TestRunLocalHSLMatchingTreatsDamAsOnePiece(t *testing.T) {
	dress := wardrobeItem("dam")
	shoes := wardrobeItem("giay")

	res := RunLocalHSLMatching([]types.CandidateForPrompt{
		{Item: dress},
		{Item: shoes},
	}, dto.RecommendOutfitReq{})

	if len(res.Items) == 0 || res.Items[0].Role != "dam" {
		t.Fatalf("expected first role dam, got %+v", res.Items)
	}
}

func wardrobeItem(slug string) *entities.WardrobeItem {
	name := slug
	return &entities.WardrobeItem{
		AuditableEntity: entities.AuditableEntity{BaseEntity: entities.BaseEntity{ID: uuid.New()}},
		FashionItem:     &entities.FashionItem{Category: &entities.Category{Slug: slug, Name: name}},
	}
}
