package mapper

import (
	"encoding/json"
	"testing"
	"time"

	plankind "smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/plankind"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"

	"github.com/google/uuid"
)

func premiumPlan() *entities.SubscriptionPlan {
	days := 30
	return &entities.SubscriptionPlan{
		AuditableEntity: entities.AuditableEntity{
			BaseEntity: entities.BaseEntity{ID: uuid.MustParse("cb591a26-9f4a-4e86-b489-8d195c80521e")},
		},
		Slug:               "premium-monthly",
		Name:               "Premium Plan",
		AiOutfitDailyQuota: 30,
		AiChatDailyQuota:   10,
		DurationDays:       &days,
		PlanKind:           plankind.Finite,
		TierRank:           1,
		AICostPolicyID:     uuid.MustParse("aa000000-0000-0000-0000-000000000002"),
	}
}

func freePlan() *entities.SubscriptionPlan {
	return &entities.SubscriptionPlan{
		AuditableEntity: entities.AuditableEntity{
			BaseEntity: entities.BaseEntity{ID: uuid.MustParse("ea78546b-f458-47df-bc53-ea849fb75630")},
		},
		Slug:               "free",
		Name:               "Free Plan",
		AiOutfitDailyQuota: 5,
		AiChatDailyQuota:   3,
		DurationDays:       nil,
		PlanKind:           plankind.DefaultFree,
		TierRank:           0,
		AICostPolicyID:     uuid.MustParse("aa000000-0000-0000-0000-000000000001"),
	}
}

func sampleEvent(eventType string, effectiveAt time.Time) *entities.UserSubscriptionEvent {
	toPlan, toRank := "premium-monthly", 1
	fromPlan, fromRank := "free", 0
	return &entities.UserSubscriptionEvent{
		AuditableEntity: entities.AuditableEntity{
			BaseEntity: entities.BaseEntity{ID: uuid.MustParse("11111111-2222-3333-4444-555555555555")},
		},
		UserID:       uuid.MustParse("ad11ad11-ad11-ad11-ad11-ad11ad11ad12"),
		EventType:    eventType,
		FromPlanCode: &fromPlan,
		FromTierRank: &fromRank,
		ToPlanCode:   &toPlan,
		ToTierRank:   &toRank,
		OccurredAt:   effectiveAt,
		EffectiveAt:  effectiveAt,
	}
}

func sampleSub(expiresAt *time.Time) *entities.UserSubscription {
	return &entities.UserSubscription{
		UserID:    uuid.MustParse("ad11ad11-ad11-ad11-ad11-ad11ad11ad12"),
		ExpiresAt: expiresAt,
	}
}

// TestPayloadCarriesAllFourteenFields is the contract test. Every key must be
// present in the serialised JSON even when empty: the receiver distinguishes
// "absent because the plan has no end date" from "absent because this is a
// version-1 payload", and those two lead to completely different handling.
func TestPayloadCarriesAllFourteenFields(t *testing.T) {
	now := time.Date(2026, 9, 29, 8, 15, 0, 123456789, time.UTC)
	expires := now.AddDate(0, 0, 30)
	event := sampleEvent("EXTENDED", now)
	sub := sampleSub(&expires)

	payload := BuildPlanChangedPayload(event, sub, premiumPlan())
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	expected := []string{
		"eventVersion", "userId", "eventType", "toPlanId", "toPlanCode",
		"toTierRank", "aiCostPolicyId", "occurredAt", "effectiveAt",
		"effectiveTo", "durationDays", "sourceEventId",
		"sourceDepositTransactionId", "fromPlanCode",
	}
	if len(decoded) != len(expected) {
		t.Fatalf("expected exactly %d fields, got %d: %s", len(expected), len(decoded), raw)
	}
	for _, key := range expected {
		if _, ok := decoded[key]; !ok {
			t.Fatalf("missing key %q in %s", key, raw)
		}
	}
}

// TestEventTypeComesFromTheHistoryRow is FR-019. The type is read from the row,
// never from a call site, so the two sides cannot disagree about what happened.
func TestEventTypeComesFromTheHistoryRow(t *testing.T) {
	now := time.Now().UTC()
	for _, eventType := range []string{
		"ACTIVATED", "EXTENDED", "UPGRADED", "OVERLAID",
		"LIFETIME_FALLBACK_RESTORED", "SUBSCRIPTION_DOWNGRADED_TO_FREE",
		"AUTO_RENEW_UPDATED",
	} {
		payload := BuildPlanChangedPayload(sampleEvent(eventType, now), sampleSub(nil), premiumPlan())
		if payload.EventType != eventType {
			t.Fatalf("eventType was rewritten: got %q, want %q", payload.EventType, eventType)
		}
	}
}

func TestVersionIsTwo(t *testing.T) {
	payload := BuildPlanChangedPayload(sampleEvent("ACTIVATED", time.Now().UTC()), sampleSub(nil), premiumPlan())
	if payload.EventVersion != 2 {
		t.Fatalf("expected contract version 2, got %d", payload.EventVersion)
	}
}

// TestToPlanIDIsPresent pins the field that makes a grant writable at all:
// ai_policy_grants.plan_id is NOT NULL and its foreign key was dropped, so the
// receiver has no way to recover it.
func TestToPlanIDIsPresent(t *testing.T) {
	plan := premiumPlan()
	payload := BuildPlanChangedPayload(sampleEvent("ACTIVATED", time.Now().UTC()), sampleSub(nil), plan)
	if payload.ToPlanID != plan.ID.String() {
		t.Fatalf("toPlanId must be the plan's own id, got %q", payload.ToPlanID)
	}
	if payload.AICostPolicyID != plan.AICostPolicyID.String() {
		t.Fatalf("aiCostPolicyId must be the plan's policy id, got %q", payload.AICostPolicyID)
	}
}

// TestOpenEndedPlanLeavesExpiryAndDurationEmpty covers the free and lifetime
// plans. Both fields must be present and empty rather than absent.
func TestOpenEndedPlanLeavesExpiryAndDurationEmpty(t *testing.T) {
	now := time.Now().UTC()
	event := sampleEvent("ACTIVATED", now)
	toPlan, toRank := "free", 0
	event.ToPlanCode = &toPlan
	event.ToTierRank = &toRank

	payload := BuildPlanChangedPayload(event, sampleSub(nil), freePlan())
	if payload.EffectiveTo != "" {
		t.Fatalf("an open-ended plan must report an empty effectiveTo, got %q", payload.EffectiveTo)
	}
	if payload.DurationDays != 0 {
		t.Fatalf("an open-ended plan must report durationDays 0, got %d", payload.DurationDays)
	}
	raw, _ := json.Marshal(payload)
	var decoded map[string]any
	_ = json.Unmarshal(raw, &decoded)
	for _, key := range []string{"effectiveTo", "durationDays", "sourceDepositTransactionId"} {
		if _, ok := decoded[key]; !ok {
			t.Fatalf("key %q must be present even when empty: %s", key, raw)
		}
	}
}

// TestDurationIsCarriedForARenewal is what makes the renewal-window rule
// computable. Without it the receiver cannot place the new period.
func TestDurationIsCarriedForARenewal(t *testing.T) {
	now := time.Now().UTC()
	expires := now.AddDate(0, 0, 60)
	payload := BuildPlanChangedPayload(sampleEvent("EXTENDED", now), sampleSub(&expires), premiumPlan())
	if payload.DurationDays != 30 {
		t.Fatalf("expected 30 duration days, got %d", payload.DurationDays)
	}
	if payload.EffectiveTo == "" {
		t.Fatal("a finite plan must carry effectiveTo")
	}
}

// TestManualGrantHasNoDepositTransaction covers the administrator path: the
// column is a foreign key into deposit_transactions and there was no payment, so
// it must be empty rather than a zero uuid.
func TestManualGrantHasNoDepositTransaction(t *testing.T) {
	payload := BuildPlanChangedPayload(sampleEvent("ACTIVATED", time.Now().UTC()), sampleSub(nil), premiumPlan())
	if payload.SourceDepositTransactionID != "" {
		t.Fatalf("a manual grant has no payment, got %q", payload.SourceDepositTransactionID)
	}
}

func TestPaymentCarriesDepositTransaction(t *testing.T) {
	depositID := uuid.MustParse("99999999-8888-7777-6666-555555555555")
	event := sampleEvent("ACTIVATED", time.Now().UTC())
	event.SourceDepositTransactionID = &depositID

	payload := BuildPlanChangedPayload(event, sampleSub(nil), premiumPlan())
	if payload.SourceDepositTransactionID != depositID.String() {
		t.Fatalf("expected the deposit id, got %q", payload.SourceDepositTransactionID)
	}
}

// TestFirstGrantHasNoFromPlanCode: an account that never had a plan must not
// claim one, because the receiver would otherwise see a phantom downgrade.
func TestFirstGrantHasNoFromPlanCode(t *testing.T) {
	event := sampleEvent("ACTIVATED", time.Now().UTC())
	event.FromPlanCode = nil

	payload := BuildPlanChangedPayload(event, sampleSub(nil), premiumPlan())
	if payload.FromPlanCode != "" {
		t.Fatalf("expected an empty fromPlanCode, got %q", payload.FromPlanCode)
	}
}

// TestTimestampsAreUTCAndRFC3339Nano: the receiver parses RFC3339Nano and
// compares instants, so a local-time timestamp would shift the ordering check.
func TestTimestampsAreUTCAndRFC3339Nano(t *testing.T) {
	local := time.Date(2026, 9, 29, 15, 30, 0, 0, time.FixedZone("ICT", 7*3600))
	payload := BuildPlanChangedPayload(sampleEvent("ACTIVATED", local), sampleSub(nil), premiumPlan())

	parsed, err := time.Parse(time.RFC3339Nano, payload.OccurredAt)
	if err != nil {
		t.Fatalf("occurredAt is not RFC3339Nano: %v", err)
	}
	if !parsed.Equal(local) {
		t.Fatalf("the instant changed during formatting: %v vs %v", parsed, local)
	}
	if parsed.Location() != time.UTC {
		t.Fatalf("timestamps must be emitted in UTC, got %v", parsed.Location())
	}
}
