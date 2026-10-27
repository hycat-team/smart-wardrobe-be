package completion

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	uc_interfaces "smart-wardrobe-be/services/commerce/internal/modules/fashion/application/interface/usecase"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/domain/entities"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/domain/repositories"
	wardrobeDto "smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/dto"
	wardrobe_contract "smart-wardrobe-be/services/commerce/internal/modules/wardrobe/presentation/contract"
	"smart-wardrobe-be/shared/application/constants/apperror"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// nopCompletionLogger keeps the fail-soft warning lines out of the test output.
type nopCompletionLogger struct{}

func (nopCompletionLogger) Debug(string, ...zap.Field) {}
func (nopCompletionLogger) Info(string, ...zap.Field)  {}
func (nopCompletionLogger) Warn(string, ...zap.Field)  {}
func (nopCompletionLogger) Error(string, ...zap.Field) {}
func (nopCompletionLogger) Fatal(string, ...zap.Field) {}

// fakeCompletionWardrobeContract serves the active-item ownership read.
type fakeCompletionWardrobeContract struct {
	wardrobe_contract.IWardrobeContract
	activeItems []*wardrobeDto.WardrobeItemContractDTO
	activeErr   error
	activeCalls int
}

func (f *fakeCompletionWardrobeContract) GetActiveWardrobeItemsByUserID(
	context.Context,
	uuid.UUID,
	int,
) ([]*wardrobeDto.WardrobeItemContractDTO, error) {
	f.activeCalls++
	return f.activeItems, f.activeErr
}

// fakeCompletionFashionRepo serves the category metadata used for role mapping.
type fakeCompletionFashionRepo struct {
	repositories.IFashionItemRepository
	items map[uuid.UUID]*entities.FashionItem
	err   error
	calls int
}

func (f *fakeCompletionFashionRepo) GetByIDs(_ context.Context, ids []uuid.UUID) ([]*entities.FashionItem, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	items := make([]*entities.FashionItem, 0, len(ids))
	for _, id := range ids {
		if item := f.items[id]; item != nil {
			items = append(items, item)
		}
	}
	return items, nil
}

// fakeCompletionSuggestionBuilder records exactly what the completion use case
// asks the shared US2 builder for.
type fakeCompletionSuggestionBuilder struct {
	calls    int
	userID   uuid.UUID
	input    dto.RecommendOutfitReq
	gapRoles []string
	returned []dto.OutfitSuggestionRes
}

func (f *fakeCompletionSuggestionBuilder) BuildOutfitCompletionSuggestions(
	_ context.Context,
	userID uuid.UUID,
	input dto.RecommendOutfitReq,
	gapRoles []string,
) []dto.OutfitSuggestionRes {
	f.calls++
	f.userID = userID
	f.input = input
	f.gapRoles = append([]string(nil), gapRoles...)
	return f.returned
}

// ownedFashionItem builds a wardrobe item the user owns plus its fashion row.
func ownedFashionItem(slug string) (*wardrobeDto.WardrobeItemContractDTO, *entities.FashionItem) {
	fashionItem := &entities.FashionItem{Category: &entities.Category{Slug: slug}}
	fashionItem.ID = uuid.New()
	wardrobeItem := &wardrobeDto.WardrobeItemContractDTO{
		ID:            uuid.New(),
		FashionItemID: &fashionItem.ID,
	}
	return wardrobeItem, fashionItem
}

// newCompletionUseCase wires the use case with in-memory fakes. There is
// deliberately no quota dependency: the constructor cannot obtain one, which is
// the compile-time proof that this endpoint can never consume a daily AI outfit
// attempt (FR-024).
func newCompletionUseCase(
	activeItems []*wardrobeDto.WardrobeItemContractDTO,
	fashionItems map[uuid.UUID]*entities.FashionItem,
	builder *fakeCompletionSuggestionBuilder,
) uc_interfaces.IOutfitCompletionUseCase {
	return NewOutfitCompletionUseCase(
		nopCompletionLogger{},
		&fakeCompletionWardrobeContract{activeItems: activeItems},
		&fakeCompletionFashionRepo{items: fashionItems},
		builder,
	)
}

func TestCompleteOutfitRejectsItemNotOwned(t *testing.T) {
	top, topFashion := ownedFashionItem("ao")
	foreignFashionID := uuid.New()
	builder := &fakeCompletionSuggestionBuilder{}

	uc := newCompletionUseCase(
		[]*wardrobeDto.WardrobeItemContractDTO{top},
		map[uuid.UUID]*entities.FashionItem{topFashion.ID: topFashion},
		builder,
	)

	_, err := uc.CompleteOutfit(context.Background(), uuid.New(), dto.CompleteOutfitReq{
		FashionItemIDs: []uuid.UUID{topFashion.ID, foreignFashionID},
	})
	if err == nil {
		t.Fatal("expected a rejection for an item the user does not own")
	}
	appErr := apperror.From(err)
	if appErr.Status != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (contracts §2.3)", appErr.Status)
	}
	if !strings.Contains(appErr.Message, foreignFashionID.String()) {
		t.Fatalf("message %q should name the offending item", appErr.Message)
	}
	if builder.calls != 0 {
		t.Fatal("no suggestion must be built for a rejected request")
	}
}

func TestCompleteOutfitSuggestsMissingRole(t *testing.T) {
	top, topFashion := ownedFashionItem("ao")
	bottom, bottomFashion := ownedFashionItem("quan")
	builder := &fakeCompletionSuggestionBuilder{
		returned: []dto.OutfitSuggestionRes{{
			Role:        "footwear",
			Reason:      "Tủ đồ chưa có giày phù hợp để hoàn thiện bộ này.",
			FashionItem: &dto.RecommendedFashionItemBriefRes{ID: uuid.New()},
		}},
	}
	uc := newCompletionUseCase(
		[]*wardrobeDto.WardrobeItemContractDTO{top, bottom},
		map[uuid.UUID]*entities.FashionItem{topFashion.ID: topFashion, bottomFashion.ID: bottomFashion},
		builder,
	)

	res, err := uc.CompleteOutfit(context.Background(), uuid.New(), dto.CompleteOutfitReq{
		FashionItemIDs: []uuid.UUID{topFashion.ID, bottomFashion.ID},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := []string{"footwear"}; !reflect.DeepEqual(builder.gapRoles, want) {
		t.Fatalf("gap roles = %v, want %v", builder.gapRoles, want)
	}
	if len(res.Suggestions) != 1 || res.Suggestions[0].Role != "footwear" {
		t.Fatalf("suggestions = %+v, want the footwear suggestion", res.Suggestions)
	}
}

func TestCompleteOutfitReturnsEmptyWhenStructureComplete(t *testing.T) {
	top, topFashion := ownedFashionItem("ao")
	bottom, bottomFashion := ownedFashionItem("quan")
	shoes, shoesFashion := ownedFashionItem("giay")
	builder := &fakeCompletionSuggestionBuilder{}

	uc := newCompletionUseCase(
		[]*wardrobeDto.WardrobeItemContractDTO{top, bottom, shoes},
		map[uuid.UUID]*entities.FashionItem{
			topFashion.ID:    topFashion,
			bottomFashion.ID: bottomFashion,
			shoesFashion.ID:  shoesFashion,
		},
		builder,
	)

	res, err := uc.CompleteOutfit(context.Background(), uuid.New(), dto.CompleteOutfitReq{
		FashionItemIDs: []uuid.UUID{topFashion.ID, bottomFashion.ID, shoesFashion.ID},
	})
	if err != nil {
		t.Fatalf("a complete structure must not fail: %v", err)
	}
	if res.Suggestions == nil {
		t.Fatal("suggestions must be an empty array, not null (FR-025)")
	}
	if len(res.Suggestions) != 0 {
		t.Fatalf("suggestions = %+v, want none", res.Suggestions)
	}
	if builder.calls != 0 {
		t.Fatal("the builder must not be called when nothing is missing")
	}
}

func TestCompleteOutfitFullbodyReplacesSeparates(t *testing.T) {
	dress, dressFashion := ownedFashionItem("dam")
	shoes, shoesFashion := ownedFashionItem("giay")
	builder := &fakeCompletionSuggestionBuilder{}

	uc := newCompletionUseCase(
		[]*wardrobeDto.WardrobeItemContractDTO{dress, shoes},
		map[uuid.UUID]*entities.FashionItem{dressFashion.ID: dressFashion, shoesFashion.ID: shoesFashion},
		builder,
	)

	res, err := uc.CompleteOutfit(context.Background(), uuid.New(), dto.CompleteOutfitReq{
		FashionItemIDs: []uuid.UUID{dressFashion.ID, shoesFashion.ID},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Suggestions) != 0 || builder.calls != 0 {
		t.Fatalf("a dress plus footwear is structurally complete, got %+v", res.Suggestions)
	}
}

func TestCompleteOutfitRequestsOuterwearForColdContext(t *testing.T) {
	top, topFashion := ownedFashionItem("ao")
	bottom, bottomFashion := ownedFashionItem("quan")
	shoes, shoesFashion := ownedFashionItem("giay")
	builder := &fakeCompletionSuggestionBuilder{
		returned: []dto.OutfitSuggestionRes{{Role: "outerwear", Reason: "..."}},
	}
	uc := newCompletionUseCase(
		[]*wardrobeDto.WardrobeItemContractDTO{top, bottom, shoes},
		map[uuid.UUID]*entities.FashionItem{
			topFashion.ID:    topFashion,
			bottomFashion.ID: bottomFashion,
			shoesFashion.ID:  shoesFashion,
		},
		builder,
	)

	weather := "cold"
	res, err := uc.CompleteOutfit(context.Background(), uuid.New(), dto.CompleteOutfitReq{
		FashionItemIDs: []uuid.UUID{topFashion.ID, bottomFashion.ID, shoesFashion.ID},
		Weather:        &weather,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := []string{"outerwear"}; !reflect.DeepEqual(builder.gapRoles, want) {
		t.Fatalf("gap roles = %v, want %v", builder.gapRoles, want)
	}
	if builder.input.Weather == nil || *builder.input.Weather != weather {
		t.Fatalf("builder must receive the request context, got %+v", builder.input)
	}
	if len(res.Suggestions) != 1 || res.Suggestions[0].Role != "outerwear" {
		t.Fatalf("suggestions = %+v, want the outerwear suggestion", res.Suggestions)
	}
}

// TestCompleteOutfitResponseShape pins that the response carries suggestions
// and nothing else: there is no remainingQuota field, so a client cannot even
// observe a quota change from this endpoint (SC-012).
func TestCompleteOutfitResponseShape(t *testing.T) {
	payload, err := json.Marshal(dto.CompleteOutfitRes{Suggestions: []dto.OutfitSuggestionRes{}})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(payload) != `{"suggestions":[]}` {
		t.Fatalf("response JSON = %s, want only an empty suggestions array", payload)
	}
}

func TestCompletionGapRoles(t *testing.T) {
	weather := func(value string) *string { return &value }

	cases := []struct {
		name    string
		present []string
		weather *string
		want    []string
	}{
		{
			name:    "top only needs bottom and footwear",
			present: []string{"top"},
			want:    []string{"bottom", "footwear"},
		},
		{
			name:    "fullbody replaces top and bottom",
			present: []string{"fullbody"},
			want:    []string{"footwear"},
		},
		{
			name:    "complete structure has no gap",
			present: []string{"top", "bottom", "footwear"},
			want:    nil,
		},
		{
			name:    "cold context adds outerwear",
			present: []string{"top", "bottom", "footwear"},
			weather: weather("cold"),
			want:    []string{"outerwear"},
		},
		{
			name:    "hot context never adds outerwear",
			present: []string{"top", "bottom", "footwear"},
			weather: weather("hot"),
			want:    nil,
		},
		{
			name:    "outerwear already present is not a gap",
			present: []string{"top", "bottom", "footwear", "outerwear"},
			weather: weather("cold"),
			want:    nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			present := make(map[string]bool, len(tc.present))
			for _, role := range tc.present {
				present[role] = true
			}
			got := completionGapRoles(present, dto.RecommendOutfitReq{Weather: tc.weather})
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("gap roles = %v, want %v", got, tc.want)
			}
		})
	}
}
