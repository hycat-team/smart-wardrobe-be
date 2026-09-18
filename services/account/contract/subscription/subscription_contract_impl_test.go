package contract

import (
	"context"
	"testing"
	"time"

	"smart-wardrobe-be/services/account/dto/subscription"
	subscriptioncaching "smart-wardrobe-be/services/account/internal/modules/subscription/application/interface/caching"
	uc_interfaces "smart-wardrobe-be/services/account/internal/modules/subscription/application/interface/usecase"

	"github.com/google/uuid"
)

// fakeOverviewCache lets each test script the cache outcome and observe writes.
type fakeOverviewCache struct {
	entry    *subscriptioncaching.SubscriptionOverviewCacheEntry
	getErr   error
	sets     int
	invalids int
}

func (f *fakeOverviewCache) Get(ctx context.Context, userID uuid.UUID) (*subscriptioncaching.SubscriptionOverviewCacheEntry, error) {
	return f.entry, f.getErr
}
func (f *fakeOverviewCache) Set(ctx context.Context, userID uuid.UUID, entry *subscriptioncaching.SubscriptionOverviewCacheEntry) error {
	f.sets++
	return nil
}
func (f *fakeOverviewCache) Invalidate(ctx context.Context, userID uuid.UUID) error {
	f.invalids++
	return nil
}

// fakeSubscriptionUC embeds the interface so only the overview method is
// scripted; every other method panics if accidentally called.
type fakeSubscriptionUC struct {
	uc_interfaces.ISubscriptionUseCase
	calls  int
	result *dto.UserSubscriptionOverviewRes
	err    error
}

func (f *fakeSubscriptionUC) GetUserSubscriptionOverview(ctx context.Context, userID uuid.UUID) (*dto.UserSubscriptionOverviewRes, error) {
	f.calls++
	return f.result, f.err
}

func overviewTestRes() *dto.UserSubscriptionOverviewRes {
	expires := time.Now().Add(72 * time.Hour).UTC()
	return &dto.UserSubscriptionOverviewRes{
		PlanID:             uuid.New(),
		PlanName:           "Premium",
		PlanSlug:           "premium",
		ExpiresAt:          &expires,
		IsAutoRenewEnabled: true,
		MaxWardrobeItems:   100,
		MaxOutfits:         50,
		AiOutfitDailyQuota: 10,
		AiChatDailyQuota:   20,
	}
}

func overviewTestContract(t *testing.T, cache *fakeOverviewCache, uc *fakeSubscriptionUC) IUserSubscriptionContract {
	t.Helper()
	return NewUserSubscriptionContract(uc, cache)
}

// Cache hit: the use case must not be called and the cached entry is served.
func TestGetUserSubscriptionOverviewServesFromCacheOnHit(t *testing.T) {
	entry := overviewContractEntryFromRes(overviewTestRes())
	cache := &fakeOverviewCache{entry: entry}
	uc := &fakeSubscriptionUC{calls: 0}
	contract := overviewTestContract(t, cache, uc)

	res, err := contract.GetUserSubscriptionOverview(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if uc.calls != 0 {
		t.Fatalf("expected use case to be skipped on cache hit, got %d calls", uc.calls)
	}
	if res.PlanSlug != "premium" || res.AiOutfitDailyQuota != 10 {
		t.Fatalf("expected cached payload served, got %+v", res)
	}
	if cache.sets != 0 {
		t.Fatalf("expected no cache fill on hit, got %d", cache.sets)
	}
}

// Cache miss: the use case is called exactly once and the cache is filled.
func TestGetUserSubscriptionOverviewFillsCacheOnMiss(t *testing.T) {
	cache := &fakeOverviewCache{}
	uc := &fakeSubscriptionUC{result: overviewTestRes()}
	contract := overviewTestContract(t, cache, uc)

	res, err := contract.GetUserSubscriptionOverview(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if uc.calls != 1 {
		t.Fatalf("expected exactly one use case call on miss, got %d", uc.calls)
	}
	if cache.sets != 1 {
		t.Fatalf("expected cache filled on miss, got %d sets", cache.sets)
	}
	if res.PlanSlug != "premium" {
		t.Fatalf("expected use case payload served, got %+v", res)
	}
}

// Use case failure: the error propagates and the cache must not be filled.
func TestGetUserSubscriptionOverviewDoesNotFillCacheOnError(t *testing.T) {
	cache := &fakeOverviewCache{}
	uc := &fakeSubscriptionUC{err: context.DeadlineExceeded}
	contract := overviewTestContract(t, cache, uc)

	if _, err := contract.GetUserSubscriptionOverview(context.Background(), uuid.New()); err == nil {
		t.Fatal("expected use case error to propagate, got nil")
	}
	if cache.sets != 0 {
		t.Fatalf("expected no cache fill on use case error, got %d", cache.sets)
	}
}

// Cache backend failure: degrade to a miss and serve from the use case.
func TestGetUserSubscriptionOverviewFallsBackOnCacheError(t *testing.T) {
	cache := &fakeOverviewCache{getErr: context.DeadlineExceeded}
	uc := &fakeSubscriptionUC{result: overviewTestRes()}
	contract := overviewTestContract(t, cache, uc)

	res, err := contract.GetUserSubscriptionOverview(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if uc.calls != 1 || cache.sets != 1 {
		t.Fatalf("expected fallback to use case with cache fill, got calls=%d sets=%d", uc.calls, cache.sets)
	}
	if res.PlanSlug != "premium" {
		t.Fatalf("expected use case payload served, got %+v", res)
	}
}

func overviewContractEntryFromRes(res *dto.UserSubscriptionOverviewRes) *subscriptioncaching.SubscriptionOverviewCacheEntry {
	expires := time.Now().Add(72 * time.Hour).UTC()
	return &subscriptioncaching.SubscriptionOverviewCacheEntry{
		PlanID:             res.PlanID,
		PlanName:           res.PlanName,
		PlanSlug:           res.PlanSlug,
		ExpiresAt:          &expires,
		IsAutoRenewEnabled: res.IsAutoRenewEnabled,
		MaxWardrobeItems:   res.MaxWardrobeItems,
		MaxOutfits:         res.MaxOutfits,
		AiOutfitDailyQuota: res.AiOutfitDailyQuota,
		AiChatDailyQuota:   res.AiChatDailyQuota,
	}
}
