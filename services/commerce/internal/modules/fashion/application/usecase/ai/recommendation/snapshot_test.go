package recommendation

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	uc_caching "smart-wardrobe-be/services/commerce/internal/modules/fashion/application/interface/caching"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/usecase/ai/recommendation/types"

	"github.com/google/uuid"
)

// fakeSnapshotCache records the snapshots the workflow writes.
type fakeSnapshotCache struct {
	sets       int
	snapshot   *uc_caching.RecommendationSnapshot
	writtenFor uuid.UUID
	setErr     error
}

func (c *fakeSnapshotCache) Set(
	_ context.Context,
	id uuid.UUID,
	snapshot *uc_caching.RecommendationSnapshot,
) error {
	c.sets++
	c.writtenFor = id
	c.snapshot = snapshot
	return c.setErr
}

func (c *fakeSnapshotCache) Get(context.Context, uuid.UUID) (*uc_caching.RecommendationSnapshot, error) {
	return nil, nil
}

func (c *fakeSnapshotCache) Delete(context.Context, uuid.UUID) error { return nil }

func snapshotItem(fashionItemID uuid.UUID) *dto.RecommendedItemRes {
	return &dto.RecommendedItemRes{
		ID:          uuid.New(),
		FashionItem: &dto.RecommendedFashionItemBriefRes{ID: fashionItemID},
	}
}

func snapshotTestFilter(explicit, learned []string) *candidateFilterResult {
	return &candidateFilterResult{
		Personalization: types.PersonalizationContext{
			Active:         true,
			ExplicitStyles: explicit,
			LearnedStyles:  learned,
		},
	}
}

func TestRecordRecommendationSnapshotStoresDedupedFashionItems(t *testing.T) {
	cache := &fakeSnapshotCache{}
	uc := &OutfitRecommendationUseCase{logger: nopRecommendationLogger{}, snapshotCache: cache}
	userID := uuid.New()
	recommendationID := uuid.New()
	first, second := uuid.New(), uuid.New()
	now := time.Now()

	finalRes := &dto.RecommendedOutfitRes{
		Items: []*dto.RecommendedItemGroup{
			{
				Primary: snapshotItem(first),
				Alternatives: []*dto.RecommendedItemRes{
					snapshotItem(second),
					{ID: uuid.New(), ItemContext: "brand_item"}, // brand item: no fashion item
				},
			},
			{
				Primary:      snapshotItem(first), // duplicate across groups
				Alternatives: []*dto.RecommendedItemRes{snapshotItem(uuid.Nil)},
			},
			nil,
		},
	}

	uc.recordRecommendationSnapshot(
		context.Background(), userID, recommendationID, finalRes,
		snapshotTestFilter([]string{"minimalist"}, []string{"Minimalist", "vintage", " "}),
		now,
	)

	if cache.sets != 1 {
		t.Fatalf("snapshot sets = %d, want 1", cache.sets)
	}
	if cache.writtenFor != recommendationID {
		t.Fatalf("written for %s, want %s", cache.writtenFor, recommendationID)
	}
	if cache.snapshot.UserID != userID {
		t.Fatalf("snapshot user = %s, want %s", cache.snapshot.UserID, userID)
	}
	if want := []uuid.UUID{first, second}; !reflect.DeepEqual(cache.snapshot.FashionItemIDs, want) {
		t.Fatalf("item ids = %v, want %v", cache.snapshot.FashionItemIDs, want)
	}
	if want := []string{"minimalist", "vintage"}; !reflect.DeepEqual(cache.snapshot.Styles, want) {
		t.Fatalf("styles = %v, want %v", cache.snapshot.Styles, want)
	}
	if !cache.snapshot.CreatedAt.Equal(now.UTC()) {
		t.Fatalf("createdAt = %s, want %s", cache.snapshot.CreatedAt, now.UTC())
	}
}

func TestRecordRecommendationSnapshotSkipsEmptyOutfit(t *testing.T) {
	cache := &fakeSnapshotCache{}
	uc := &OutfitRecommendationUseCase{logger: nopRecommendationLogger{}, snapshotCache: cache}

	uc.recordRecommendationSnapshot(
		context.Background(), uuid.New(), uuid.New(),
		&dto.RecommendedOutfitRes{Items: []*dto.RecommendedItemGroup{
			{Primary: &dto.RecommendedItemRes{ID: uuid.New(), ItemContext: "brand_item"}},
		}},
		snapshotTestFilter(nil, nil),
		time.Now(),
	)

	if cache.sets != 0 {
		t.Fatalf("snapshot sets = %d, want 0: an empty item list is not stored", cache.sets)
	}
}

func TestRecordRecommendationSnapshotToleratesCacheFailure(t *testing.T) {
	cache := &fakeSnapshotCache{setErr: errors.New("redis down")}
	uc := &OutfitRecommendationUseCase{logger: nopRecommendationLogger{}, snapshotCache: cache}

	uc.recordRecommendationSnapshot(
		context.Background(), uuid.New(), uuid.New(),
		&dto.RecommendedOutfitRes{Items: []*dto.RecommendedItemGroup{
			{Primary: snapshotItem(uuid.New())},
		}},
		snapshotTestFilter(nil, nil),
		time.Now(),
	)

	if cache.sets != 1 {
		t.Fatalf("snapshot sets = %d, want 1", cache.sets)
	}
}

func TestRecordRecommendationSnapshotWithoutCacheIsNoOp(t *testing.T) {
	uc := &OutfitRecommendationUseCase{logger: nopRecommendationLogger{}}

	uc.recordRecommendationSnapshot(
		context.Background(), uuid.New(), uuid.New(),
		&dto.RecommendedOutfitRes{Items: []*dto.RecommendedItemGroup{
			{Primary: snapshotItem(uuid.New())},
		}},
		snapshotTestFilter(nil, nil),
		time.Now(),
	)
}
