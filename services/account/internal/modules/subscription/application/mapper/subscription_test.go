package mapper

import (
	"encoding/json"
	"testing"
	"time"

	subscriptiondto "smart-wardrobe-be/services/account/dto/subscription"
	caching "smart-wardrobe-be/services/account/internal/modules/subscription/application/interface/caching"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"

	"github.com/google/uuid"
)

// 033 T023 (D2/D12) - the cache leg of the entitlement chain. The overview is
// cached for 60 seconds, so a cache entry written before this feature has no
// `personalizationEnabled` key at all. Decoding it must yield false, which
// keeps personalization OFF rather than failing or inventing an entitlement.

// oldCacheEntryJSON is a cache entry written by the previous release: no
// personalizationEnabled key, and the key set must survive the round-trip.
const oldCacheEntryJSON = `{
	"planId": "3f0f5f0e-9d1a-4a5f-9c2f-2f9c7a1f0b11",
	"planName": "Premium",
	"planSlug": "premium",
	"expiresAt": "2027-01-01T00:00:00Z",
	"isAutoRenewEnabled": true,
	"maxWardrobeItems": 500,
	"maxOutfits": 200,
	"aiOutfitDailyQuota": 20,
	"aiChatDailyQuota": 50,
	"cachedAt": "2026-10-20T05:30:00Z"
}`

func TestOldCacheEntryDecodesPersonalizationOff(t *testing.T) {
	var entry caching.SubscriptionOverviewCacheEntry
	if err := json.Unmarshal([]byte(oldCacheEntryJSON), &entry); err != nil {
		t.Fatalf("decode an old cache entry: %v", err)
	}
	if entry.PersonalizationEnabled {
		t.Fatal("an absent personalizationEnabled key must decode as false (033 D12)")
	}

	contract := CacheEntryToContractDTO(&entry)
	if contract == nil {
		t.Fatal("expected a contract payload for a decoded entry")
	}
	if contract.PersonalizationEnabled {
		t.Fatalf("cache entry -> contract DTO turned personalization on: %+v", contract)
	}
	if contract.PlanSlug != "premium" || contract.MaxWardrobeItems != 500 {
		t.Fatalf("the existing fields must survive: %+v", contract)
	}
}

// TestCacheEntryRoundTripKeepsPersonalization covers the write side: an
// entitlement that IS enabled must reach the cache and come back on, and the
// stored JSON must carry the camelCase key the frontend sees elsewhere.
func TestCacheEntryRoundTripKeepsPersonalization(t *testing.T) {
	contract := &subscriptiondto.UserSubscriptionContractDTO{
		PlanID:                 uuid.New(),
		PlanName:               "Premium",
		PlanSlug:               "premium",
		MaxWardrobeItems:       500,
		MaxOutfits:             200,
		AiOutfitDailyQuota:     20,
		AiChatDailyQuota:       50,
		PersonalizationEnabled: true,
	}

	entry := ContractDTOToCacheEntry(contract)
	if entry == nil {
		t.Fatal("ContractDTOToCacheEntry returned nil for a non-nil contract")
	}
	if !entry.PersonalizationEnabled {
		t.Fatal("ContractDTOToCacheEntry dropped personalizationEnabled")
	}
	if entry.CachedAt.IsZero() {
		t.Fatal("ContractDTOToCacheEntry did not stamp CachedAt")
	}

	raw, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("marshal the cache entry: %v", err)
	}
	var decoded caching.SubscriptionOverviewCacheEntry
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("decode the cache entry: %v", err)
	}

	back := CacheEntryToContractDTO(&decoded)
	if back == nil || !back.PersonalizationEnabled {
		t.Fatalf("personalizationEnabled did not survive the cache round-trip: %+v", back)
	}
}

// TestDisabledCacheEntryOmitsTheKey keeps old payloads byte-compatible: with
// `omitempty`, a disabled entitlement adds no key at all.
func TestDisabledCacheEntryOmitsTheKey(t *testing.T) {
	entry := ContractDTOToCacheEntry(&subscriptiondto.UserSubscriptionContractDTO{
		PlanID:   uuid.New(),
		PlanName: "Free",
		PlanSlug: "free",
	})

	raw, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("marshal the cache entry: %v", err)
	}
	var fields map[string]any
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatalf("decode the cache entry as a generic object: %v", err)
	}
	if _, ok := fields["personalizationEnabled"]; ok {
		t.Fatalf("a disabled entry must omit the key, got %s", string(raw))
	}
}

// TestBuildUserSubscriptionOverviewResReadsPlanEntitlement pins D2: the overview
// reads the CURRENT plan row, never the grant-time benefit snapshot, so an
// admin retuning a plan reaches users without a re-grant.
func TestBuildUserSubscriptionOverviewResReadsPlanEntitlement(t *testing.T) {
	planID := uuid.New()
	expires := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	plan := &entities.SubscriptionPlan{
		Slug:                   "premium",
		Name:                   "Premium",
		MaxWardrobeItems:       500,
		MaxOutfits:             200,
		AiOutfitDailyQuota:     20,
		AiChatDailyQuota:       50,
		PersonalizationEnabled: true,
	}
	plan.ID = planID
	sub := &entities.UserSubscription{
		UserID:             uuid.New(),
		SubscriptionPlanID: planID,
		ExpiresAt:          &expires,
		// A snapshot from before the feature carries no personalization key.
		CurrentBenefitSnapshot: entities.JSONDocument(`{"maxWardrobeItems":500}`),
	}

	res := BuildUserSubscriptionOverviewRes(sub, plan)
	if res == nil {
		t.Fatal("BuildUserSubscriptionOverviewRes returned nil")
	}
	if !res.PersonalizationEnabled {
		t.Fatal("the overview must read personalization_enabled from the plan row")
	}

	contract := MapUserSubscriptionOverviewToContractDTO(res)
	if contract == nil || !contract.PersonalizationEnabled {
		t.Fatalf("the entitlement was lost between overview and contract DTO: %+v", contract)
	}
}
