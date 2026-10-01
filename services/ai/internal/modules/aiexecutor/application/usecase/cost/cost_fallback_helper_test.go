package cost

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/application/usecase/grant"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/aienforcementmode"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/aioperation"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/airoute"
	aiexecutor_entities "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/entities"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/repositories"
	"smart-wardrobe-be/shared/pkg/logger"

	"github.com/google/uuid"
)

var testNow = time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)

// recordingCostRepo records what the degraded path touched.
type recordingCostRepo struct {
	repositories.IAICostRepository

	createdEvents []*aiexecutor_entities.AIUsageEvent
	reserveCalls  int
	createErr     error
	// 024 T130: the cap the use case resolved, so a test can assert the SNAPSHOT
	// value reaches the repository rather than the live column.
	reservedCap *int
}

func (r *recordingCostRepo) CreateEvent(_ context.Context, event *aiexecutor_entities.AIUsageEvent) error {
	if r.createErr != nil {
		return r.createErr
	}
	r.createdEvents = append(r.createdEvents, event)
	return nil
}

func (r *recordingCostRepo) Reserve(_ context.Context, _ *aiexecutor_entities.AIPolicyGrant, _ *aiexecutor_entities.AICostPolicy, _ *aiexecutor_entities.AICostPolicyOperation, cap *int, _ *aiexecutor_entities.AIUsageEvent, _ time.Time) (*aiexecutor_entities.AIUsagePeriodLedger, bool, error) {
	r.reserveCalls++
	r.reservedCap = cap
	return nil, true, nil
}

func newTier2Fixture() (*recordingCostRepo, uuid.UUID, uuid.UUID) {
	return &recordingCostRepo{}, uuid.New(), uuid.New()
}

// ---- tier 2 ----

// Tier 2 writes no ledger row: the read path must not change the
// policy-grant data (FR-042), and it must not call Reserve either.
func TestTier2WritesNoLedgerRow(t *testing.T) {
	repo, requestID, userID := newTier2Fixture()

	decision, err := buildTier2Decision(context.Background(), repo, requestID, userID,
		aioperation.Chat, "google", "gemini", 1200, testNow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.reserveCalls != 0 {
		t.Error("tier 2 must not reserve: it permits zero paid attempts, so there is nothing to reserve against")
	}
	if len(repo.createdEvents) != 1 {
		t.Fatalf("expected exactly one usage event, got %d", len(repo.createdEvents))
	}
	if repo.createdEvents[0].LedgerID != nil {
		t.Error("a tier-2 event must have a NULL ledger id - that is what keeps the ledger untouched")
	}
	if decision.Route != airoute.Free {
		t.Errorf("tier 2 must run on the free route, got %s", decision.Route)
	}
	if !decision.Tracked {
		t.Error("tier 2 must stay tracked, otherwise there is no verification data at all")
	}
}

// The degraded budget is bounded. An uncapped tier-2 request is how a fallback
// turns into an incident, so the ceiling is part of the contract with the
// orchestrator and is asserted here rather than trusted.
func TestTier2AppliesTheDegradedTokenCeiling(t *testing.T) {
	repo, requestID, userID := newTier2Fixture()

	decision, err := buildTier2Decision(context.Background(), repo, requestID, userID,
		aioperation.Chat, "google", "gemini", 1200, testNow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if decision.MaxInputTokens != tier2MaxInputTokens {
		t.Errorf("input ceiling = %d, want %d", decision.MaxInputTokens, tier2MaxInputTokens)
	}
	if decision.MaxOutputTokens != tier2MaxOutputTokens {
		t.Errorf("output ceiling = %d, want %d", decision.MaxOutputTokens, tier2MaxOutputTokens)
	}
}

// The event still records which account and operation ran, because FR-010 makes
// the degradation accountable and a row without a user cannot be attributed.
func TestTier2EventCarriesTheAccountAndOperation(t *testing.T) {
	repo, requestID, userID := newTier2Fixture()

	if _, err := buildTier2Decision(context.Background(), repo, requestID, userID,
		aioperation.Outfit, "google", "gemini", 1200, testNow); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(repo.createdEvents) != 1 {
		t.Fatalf("expected exactly one usage event, got %d", len(repo.createdEvents))
	}
	event := repo.createdEvents[0]
	if event.UserID == nil || *event.UserID != userID {
		t.Errorf("event must carry the account id, got %v", event.UserID)
	}
	if event.Operation != aioperation.Outfit {
		t.Errorf("event must carry the operation, got %s", event.Operation)
	}
	if event.RequestID != requestID {
		t.Errorf("event must carry the request id, got %s", event.RequestID)
	}
	if event.LogicalRoute != airoute.Free {
		t.Errorf("event must record the free route, got %s", event.LogicalRoute)
	}
}

// A failure to write the verification row is returned, not swallowed: the tier-2
// decision itself is free to produce, but losing the record would make the
// degradation invisible.
func TestTier2PropagatesAWriteFailure(t *testing.T) {
	repo, requestID, userID := newTier2Fixture()
	repo.createErr = errTier2Write

	if _, err := buildTier2Decision(context.Background(), repo, requestID, userID,
		aioperation.Chat, "google", "gemini", 1200, testNow); err == nil {
		t.Error("phai tra loi khi khong ghi duoc ban ghi xac nhan")
	}
}

// ---- 024 T130 / T138: the per-day cap reaches the repository from the SNAPSHOT ----

// FR-029b: the snapshot decides what the customer bought. Reading the live column
// instead meant an administrator's edit changed the limit for accounts that had
// already paid for a different number - and the admin API's `replayRequired`
// warning was describing something that never happened.
func TestReserveReceivesTheSnapshotAttemptCapNotTheLiveColumn(t *testing.T) {
	snapshotCap := 4
	liveCap := 99
	grantRow := snapshotGrantWithAttemptCap(t, &snapshotCap)
	policy := observeOnlyPolicyWithAttemptCap(&liveCap)

	uc := &CostUseCase{
		entitlements:    grant.NewEntitlementReader(logger.New("test", "", "info", false)),
		fallbackMetrics: grant.NewFallbackMetrics(logger.New("test", "", "info", false)),
	}

	got := uc.resolvePaidAttemptCap(grantRow, policy, operationOf(policy))
	if got == nil {
		t.Fatal("co grant thi phai lay tran tu snapshot")
	}
	if *got != snapshotCap {
		t.Errorf("phai dung gia tri snapshot = %d, co = %d", snapshotCap, *got)
	}
	if *got == liveCap {
		t.Error("khong duoc dung gia tri cot live")
	}
}

// With no grant there is no snapshot, so the live column is the only source
// available. This is the tier-1 fallback path and the requirement does not
// describe a snapshot for it, because none exists.
func TestReserveFallsBackToTheLiveColumnWhenThereIsNoGrant(t *testing.T) {
	liveCap := 7
	policy := observeOnlyPolicyWithAttemptCap(&liveCap)

	uc := &CostUseCase{entitlements: grant.NewEntitlementReader(logger.New("test", "", "info", false))}
	got := uc.resolvePaidAttemptCap(nil, policy, operationOf(policy))
	if got == nil || *got != liveCap {
		t.Errorf("khong co grant thi phai dung cot live = %d, co = %v", liveCap, got)
	}
}

// A nil cap must never be silently permissive. FR-005e calls NULL under
// observe_only a misconfiguration, and the safe reading of "no cap available" is
// "refuse the paid route", not "allow everything".
func TestReserveReportsNilCapRatherThanTreatingItAsUnlimited(t *testing.T) {
	policy := observeOnlyPolicyWithAttemptCap(nil)
	uc := &CostUseCase{entitlements: grant.NewEntitlementReader(logger.New("test", "", "info", false))}

	if got := uc.resolvePaidAttemptCap(nil, policy, operationOf(policy)); got != nil {
		t.Errorf("policy khong co han thi phai tra nil, co = %v", got)
	}
}

// The repository must not read the live column itself: it cannot see the
// snapshot, and a direct read there is exactly what this task removed.
//
// Asserted over the source because there is no other way to prove a negative
// about a code path that happens to be skipped by the tests above - the fake
// stands in for it, so the fake cannot catch a regression in the real file.
func TestRepositoryNoLongerReadsTheLiveAttemptCapColumn(t *testing.T) {
	src, err := os.ReadFile(filepath.Join(
		"..", "..", "..", "infrastructure", "persistence", "ai_cost_repo_impl.go",
	))
	if err != nil {
		t.Fatalf("khong doc duoc file: %v", err)
	}
	if bytes.Contains(src, []byte("operation.MaxPaidAttemptsPerDay")) {
		t.Error("repository van doc cot live; FR-029b yeu cau cap den tu snapshot")
	}
}

var errTier2Write = errTest("khong ghi duoc ban ghi xac nhan tier 2")

type errTest string

func (e errTest) Error() string { return string(e) }

func operationOf(policy *aiexecutor_entities.AICostPolicy) *aiexecutor_entities.AICostPolicyOperation {
	if policy == nil || len(policy.Operations) == 0 {
		return nil
	}
	return policy.Operations[0]
}

func observeOnlyPolicyWithAttemptCap(cap *int) *aiexecutor_entities.AICostPolicy {
	return &aiexecutor_entities.AICostPolicy{
		Code:            "free-default",
		EnforcementMode: aienforcementmode.ObserveOnly,
		Operations: []*aiexecutor_entities.AICostPolicyOperation{
			{Operation: aioperation.Outfit, MaxPaidAttemptsPerDay: cap},
		},
	}
}

// snapshotGrantWithAttemptCap builds a grant whose frozen entitlement carries the
// given per-day cap, mirroring what the event handler writes.
//
// The snapshot is assembled here rather than through a production helper because
// the reader only has to decode it, and the encoding is already covered by the
// grant package's own tests.
func snapshotGrantWithAttemptCap(t *testing.T, cap *int) *aiexecutor_entities.AIPolicyGrant {
	t.Helper()
	operations := []aioperation.AIOperation{aioperation.Chat, aioperation.Outfit, aioperation.Summary, aioperation.Rewriter}
	entitlements := make(map[string]grant.OperationEntitlement, len(operations))
	for _, op := range operations {
		entitlements[string(op)] = grant.OperationEntitlement{
			MaxPaidAttemptsPerDay:  cap,
			NormalMaxInputTokens:   6000,
			NormalMaxOutputTokens:  500,
			ReducedMaxInputTokens:  4000,
			ReducedMaxOutputTokens: 400,
		}
	}
	document, err := json.Marshal(&grant.PolicySnapshot{
		Envelope:     grant.SnapshotEnvelope{EventVersion: 2, OccurredAt: testNow},
		Entitlements: entitlements,
	})
	if err != nil {
		t.Fatalf("marshal snapshot: %v", err)
	}
	effectiveTo := testNow.AddDate(0, 1, 0)
	return &aiexecutor_entities.AIPolicyGrant{
		UserID:         uuid.New(),
		PolicySnapshot: aiexecutor_entities.JSONDocument(document),
		EffectiveFrom:  testNow,
		EffectiveTo:    &effectiveTo,
	}
}
