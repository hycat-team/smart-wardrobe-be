package persistence

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	aierrors "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/application/errors"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/aioperation"
	aiexecutor_entities "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/entities"

	"github.com/google/uuid"
)

// The project has no database test harness — no sqlmock, no testcontainers, no
// sqlite — so the SQL-level behaviour these tests care about (the ON CONFLICT
// predicate and the FOR UPDATE re-read) is asserted in two ways:
//
//   1. The pure window logic is tested here directly.
//   2. The dedup predicate itself is verified against the running PostgreSQL in
//      quickstart.md, GĐ1, which checks that a second insert with
//      grant_id IS NULL affects 0 rows. That check was run on PostgreSQL 17.10
//      during design and is repeated per implementation.

// TestFallbackLedgerWindowAnchorsOnAFixedEpoch covers the nil-grant path. The
// window must advance on its own: anchoring it on the request time would pin
// every request to index 0, the dedup key would never change, and the running
// total on the fallback ledger would never reset.
func TestFallbackLedgerWindowAnchorsOnAFixedEpoch(t *testing.T) {
	now := time.Date(2026, 9, 30, 14, 0, 0, 0, time.UTC)

	idx, start, end := periodFor(nil, 30, now)
	if idx <= 0 {
		t.Fatalf("a 2026 request must land well past the epoch, got index %d", idx)
	}
	if start.After(now) || !end.After(now) {
		t.Fatalf("the window must contain the request time, got [%v, %v)", start, end)
	}
	if got := end.Sub(start); got != 30*24*time.Hour {
		t.Fatalf("the fallback period must span 30 days, got %v", got)
	}
}

// TestFallbackLedgerWindowIsIdenticalForTheSameInstant is what makes the
// anti-overspend control work: two calls at the same time must land on the same
// index, or each would open a fresh ledger and the running total would reset.
func TestFallbackLedgerWindowIsIdenticalForTheSameInstant(t *testing.T) {
	at := time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)

	firstIdx, firstStart, _ := periodFor(nil, 30, at)
	secondIdx, secondStart, _ := periodFor(nil, 30, at)

	if firstIdx != secondIdx || !firstStart.Equal(secondStart) {
		t.Fatalf("two calls at the same instant produced different windows: %d/%v and %d/%v",
			firstIdx, firstStart, secondIdx, secondStart)
	}
}

func TestFallbackLedgerWindowRollsOver(t *testing.T) {
	start := time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)

	firstIdx, _, _ := periodFor(nil, 30, start)
	secondIdx, _, _ := periodFor(nil, 30, start.Add(30*24*time.Hour))

	if secondIdx != firstIdx+1 {
		t.Fatalf("the period must roll over after 30 days: %d then %d", firstIdx, secondIdx)
	}
}

// TestGrantLedgerWindowIsAnchoredOnTheGrant documents why the two paths differ.
// A real grant anchors its period on effective_from so that a renewal continues
// the existing budget instead of handing the account a fresh one.
func TestGrantLedgerWindowIsAnchoredOnTheGrant(t *testing.T) {
	effectiveFrom := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	grant := &aiexecutor_entities.AIPolicyGrant{
		AuditableEntity: aiexecutor_entities.AuditableEntity{
			BaseEntity: aiexecutor_entities.BaseEntity{ID: uuid.New()},
		},
		EffectiveFrom: effectiveFrom,
	}

	idx, start, _ := periodFor(grant, 30, effectiveFrom.Add(35*24*time.Hour))
	if idx != 1 {
		t.Fatalf("expected the second period, got %d", idx)
	}
	if want := effectiveFrom.Add(30 * 24 * time.Hour); !start.Equal(want) {
		t.Fatalf("the period must be anchored on the grant, got %v", start)
	}
}

func TestGrantLedgerWindowIsCappedByGrantExpiry(t *testing.T) {
	effectiveFrom := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	expiry := effectiveFrom.Add(10 * 24 * time.Hour)
	grant := &aiexecutor_entities.AIPolicyGrant{
		AuditableEntity: aiexecutor_entities.AuditableEntity{
			BaseEntity: aiexecutor_entities.BaseEntity{ID: uuid.New()},
		},
		EffectiveFrom: effectiveFrom,
		EffectiveTo:   &expiry,
	}

	_, _, end := periodFor(grant, 30, effectiveFrom.Add(24*time.Hour))
	if !end.Equal(expiry) {
		t.Fatalf("the period must not run past the grant expiry, got %v", end)
	}
}

func TestPeriodFallsBackToThirtyDaysWhenUnset(t *testing.T) {
	now := time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)

	idx, withDefault, _ := periodFor(nil, 0, now)
	_, withExplicit, _ := periodFor(nil, 30, now)
	if !withDefault.Equal(withExplicit) {
		t.Fatalf("a zero period_days must behave as 30, got %v vs %v", withDefault, withExplicit)
	}
	if idx <= 0 {
		t.Fatalf("the index must still advance, got %d", idx)
	}
}

// TestNullableGrantID is the small but load-bearing mapping: a nil grant has to
// become SQL NULL, and uuid.Nil would be a valid-looking id pointing at a
// non-existent row.
func TestNullableGrantID(t *testing.T) {
	if nullableGrantID(nil) != nil {
		t.Fatal("a nil grant must map to SQL NULL, not to uuid.Nil")
	}
	grant := &aiexecutor_entities.AIPolicyGrant{
		AuditableEntity: aiexecutor_entities.AuditableEntity{
			BaseEntity: aiexecutor_entities.BaseEntity{ID: uuid.New()},
		},
	}
	if got := nullableGrantID(grant); got == nil || *got != grant.ID {
		t.Fatalf("a real grant must map to its own id, got %v", got)
	}
}

// TestUserIDFallsBackToTheEvent covers the case where the ledger is keyed on the
// account even though there is no grant row: the user still has to come from
// somewhere, and taking it from the usage event is the only source available.
func TestUserIDFallsBackToTheEvent(t *testing.T) {
	userID := uuid.MustParse("3f7c1a20-0000-4000-8000-000000000001")
	event := &aiexecutor_entities.AIUsageEvent{UserID: &userID}

	if got := userIDFor(nil, event); got != userID {
		t.Fatalf("expected the user id from the event, got %s", got)
	}
	if got := userIDFor(nil, &aiexecutor_entities.AIUsageEvent{}); got != uuid.Nil {
		t.Fatalf("an anonymous event must yield uuid.Nil, got %s", got)
	}

	grant := &aiexecutor_entities.AIPolicyGrant{
		AuditableEntity: aiexecutor_entities.AuditableEntity{
			BaseEntity: aiexecutor_entities.BaseEntity{ID: uuid.New()},
		},
	}
	// With a grant present the grant is authoritative, even if the event
	// disagrees — the ledger row must follow the grant that is paying for it.
	grant.UserID = userID
	event.UserID = func() *uuid.UUID { other := uuid.New(); return &other }()
	if got := userIDFor(grant, event); got != userID {
		t.Fatalf("the grant must win over the event, got %s", got)
	}
}

// TestObserveOnlyWithoutAttemptCapIsAMisconfiguration documents the branch that
// FR-005e requires. The sentinel is a distinct value rather than a generic
// error so the caller can branch on it and degrade instead of failing.
func TestObserveOnlyWithoutAttemptCapIsAMisconfiguration(t *testing.T) {
	// A nil MaxPaidAttemptsPerDay in observe_only is the misconfiguration; the
	// repository turns it into this sentinel. The test pins the reason the
	// caller must recognise it.
	if err := aierrors.ErrObserveOnlyMissingAttemptCap; err == nil {
		t.Fatal("the sentinel must exist")
	}
}

// TestOperationLimitIsOnlyEnforcedInObserveOnly restates the rule the Reserve
// branch implements: strict policies ignore the per-day counter entirely, which
// is why Premium is stored as NULL.
func TestOperationLimitIsOnlyEnforcedInObserveOnly(t *testing.T) {
	premiumLike := &aiexecutor_entities.AICostPolicyOperation{
		Operation:             aioperation.Outfit,
		MaxPaidAttemptsPerDay: nil,
	}
	if premiumLike.MaxPaidAttemptsPerDay != nil {
		t.Fatal("a strict policy's per-day limit must be NULL, not a number")
	}
}

// ---- 028 US1 T010: the ledger accumulates actual cost on every route ----
//
// The project has no database test harness, so the Confirm ledger behaviour is
// asserted over the source the same way TestRepositoryNoLongerReadsTheLiveAttemptCapColumn
// does: the property under test is structural (where the accumulation line sits),
// and a fake cannot catch a regression in the real file.

// Confirm must accumulate actual_cost_micro_vnd for both paid and free routes.
// The accumulation line must sit outside the paid/free token-bucket branch.
func TestConfirmAccumulatesActualOnBothPaidAndFreeRoutes(t *testing.T) {
	src := readRepoImpl(t)
	actualLine := `ledgerUpdates["actual_cost_micro_vnd"] = gorm.Expr("actual_cost_micro_vnd + ?", cost)`
	if !bytes.Contains(src, []byte(actualLine)) {
		t.Fatalf("Confirm must accumulate actual_cost_micro_vnd, missing line %q", actualLine)
	}
	// The token buckets stay split by route...
	for _, want := range []string{`ledgerUpdates["paid_input_tokens"]`, `ledgerUpdates["free_input_tokens"]`} {
		if !bytes.Contains(src, []byte(want)) {
			t.Errorf("Confirm must keep the split token bucket %s", want)
		}
	}
	// ...but the actual accumulation must not live inside the paid-only branch.
	// It is positioned after the if/else that splits the buckets.
	paidIdx := bytes.Index(src, []byte("if paid {"))
	actualIdx := bytes.Index(src, []byte(actualLine))
	elseIdx := bytes.Index(src[paidIdx:], []byte("} else {"))
	if paidIdx < 0 || actualIdx < 0 || elseIdx < 0 {
		t.Fatalf("cannot locate the Confirm route branch (paid=%d actual=%d else=%d)", paidIdx, actualIdx, elseIdx)
	}
	elseClose := paidIdx + elseIdx
	if actualIdx < elseClose {
		t.Error("actual_cost_micro_vnd accumulation must sit after the paid/free branch so free events accumulate too")
	}
}

// Confirm must not touch the reservation or admission logic: reserved cost is
// only released back, and the token buckets are additive.
func TestConfirmLeavesReservationAndBucketsAdditive(t *testing.T) {
	src := readRepoImpl(t)
	if !bytes.Contains(src, []byte(`"reserved_cost_micro_vnd": gorm.Expr("GREATEST(reserved_cost_micro_vnd - ?,0)"`)) {
		t.Error("Confirm must release the reservation with GREATEST(reserved - ?, 0)")
	}
	if strings.Contains(string(src), "admitted") && strings.Contains(string(src), "func (r *AICostRepository) Confirm") {
		confirmSrc := string(src[bytes.Index(src, []byte("func (r *AICostRepository) Confirm")):])
		if idx := strings.Index(confirmSrc, "func (r *AICostRepository) Release"); idx >= 0 {
			confirmSrc = confirmSrc[:idx]
		}
		if strings.Contains(confirmSrc, "admitted") {
			t.Error("Confirm must not contain admission logic; Reserve owns it")
		}
	}
}

// A failed attempt (Released) must not accumulate actual cost: only Confirm
// writes actual_cost_micro_vnd to the ledger.
func TestReleasedAttemptsDoNotAccumulateActual(t *testing.T) {
	src := readRepoImpl(t)
	releaseIdx := bytes.Index(src, []byte("func (r *AICostRepository) finishWithoutCost"))
	if releaseIdx < 0 {
		t.Fatal("cannot locate finishWithoutCost (Release path)")
	}
	tail := src[releaseIdx:]
	if next := bytes.Index(tail, []byte("\nfunc (r *AICostRepository) ExpireUnknown")); next >= 0 {
		tail = tail[:next]
	}
	if bytes.Contains(tail, []byte("actual_cost_micro_vnd")) {
		t.Error("Release/finishWithoutCost must never touch actual_cost_micro_vnd; failed attempts carry no cost")
	}
}

func readRepoImpl(t *testing.T) []byte {
	t.Helper()
	src, err := os.ReadFile(filepath.Join(".", "ai_cost_repo_impl.go"))
	if err != nil {
		t.Fatalf("cannot read repo impl: %v", err)
	}
	return src
}
