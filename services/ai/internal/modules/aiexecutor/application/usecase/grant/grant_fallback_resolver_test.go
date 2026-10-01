package grant

import (
	"context"
	"errors"
	"testing"

	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/aioperation"
	aiexecutor_entities "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/entities"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/repositories"
	"smart-wardrobe-be/shared/config"
	"smart-wardrobe-be/shared/domain/constants/aifallbackreason"
	"smart-wardrobe-be/shared/pkg/logger"

	"github.com/google/uuid"
)

type fakeCostRepo struct {
	repositories.IAICostRepository

	policy     *aiexecutor_entities.AICostPolicy
	lookupErr  error
	lookupCode string
	lookups    int
}

func (f *fakeCostRepo) GetPolicyByCode(_ context.Context, code string) (*aiexecutor_entities.AICostPolicy, error) {
	f.lookups++
	f.lookupCode = code
	return f.policy, f.lookupErr
}

type recordingObserver struct {
	calls []struct {
		reason aifallbackreason.Reason
		tier2  bool
	}
}

func (r *recordingObserver) OnFallback(_ context.Context, _ uuid.UUID, _ aioperation.AIOperation, reason aifallbackreason.Reason, tier2 bool) {
	r.calls = append(r.calls, struct {
		reason aifallbackreason.Reason
		tier2  bool
	}{reason: reason, tier2: tier2})
}

func testConfig(code string) *config.Config {
	return &config.Config{AI: config.AIServiceConfig{DefaultCostPolicyCode: code}}
}

func policyWith(tiers ...aioperation.AIOperation) *aiexecutor_entities.AICostPolicy {
	p := &aiexecutor_entities.AICostPolicy{
		AuditableEntity: aiexecutor_entities.AuditableEntity{
			BaseEntity: aiexecutor_entities.BaseEntity{ID: uuid.New()},
		},
		Code:     "free-default",
		IsActive: true,
	}
	for _, tier := range tiers {
		p.Operations = append(p.Operations, &aiexecutor_entities.AICostPolicyOperation{
			AuditableEntity: aiexecutor_entities.AuditableEntity{
				BaseEntity: aiexecutor_entities.BaseEntity{ID: uuid.New()},
			},
			PolicyID:              p.ID,
			Operation:             tier,
			MaxPaidAttemptsPerDay: intPtr(4),
			IsEnabled:             true,
		})
	}
	return p
}

func intPtr(v int) *int { return &v }

var testUser = uuid.MustParse("3f7c1a20-0000-4000-8000-000000000001")

func newResolver(repo repositories.IAICostRepository, code string, observer FallbackObserver) *GrantFallbackResolver {
	return NewGrantFallbackResolver(repo, testConfig(code), logger.New("test", "", "info", false), observer)
}

// TestTier1UsedWhenDefaultPolicyIsUsable is the happy path: no grant row, but
// the configured default policy covers the operation, so tier 1 applies.
func TestTier1UsedWhenDefaultPolicyIsUsable(t *testing.T) {
	repo := &fakeCostRepo{policy: policyWith(aioperation.Chat, aioperation.Outfit)}
	observer := &recordingObserver{}
	resolver := newResolver(repo, "free-default", observer)

	got, err := resolver.Resolve(context.Background(), testUser, aioperation.Chat)
	if err != nil {
		t.Fatalf("Resolve must not fail the request: %v", err)
	}
	if got.Tier2 {
		t.Fatal("tier 1 was available, so tier 2 must not be selected")
	}
	if got.Policy == nil || got.Operation == nil {
		t.Fatalf("tier 1 must return a usable policy pair, got %+v", got)
	}
	if got.Operation.Operation != aioperation.Chat {
		t.Fatalf("resolved the wrong operation: %s", got.Operation.Operation)
	}
	if repo.lookupCode != "free-default" {
		t.Fatalf("the policy code must come from configuration, got %q", repo.lookupCode)
	}
	if len(observer.calls) != 1 || observer.calls[0].tier2 {
		t.Fatalf("expected one tier-1 observation, got %+v", observer.calls)
	}
	if observer.calls[0].reason != aifallbackreason.Tier1DefaultPolicy {
		t.Fatalf("unexpected reason code: %s", observer.calls[0].reason)
	}
}

// TestTier2WhenConfiguredCodeDoesNotExist covers a typo in
// ai.default_cost_policy_code — the most likely way to reach tier 2 in practice.
func TestTier2WhenConfiguredCodeDoesNotExist(t *testing.T) {
	repo := &fakeCostRepo{policy: nil}
	observer := &recordingObserver{}
	resolver := newResolver(repo, "does-not-exist", observer)

	got, err := resolver.Resolve(context.Background(), testUser, aioperation.Chat)
	if err != nil {
		t.Fatalf("a missing policy must degrade, not fail: %v", err)
	}
	if !got.Tier2 {
		t.Fatal("tier 2 must be selected when the configured policy is absent")
	}
	if got.Policy != nil || got.Operation != nil {
		t.Fatal("tier 2 must not pretend to have a policy")
	}
	if len(observer.calls) != 1 || !observer.calls[0].tier2 {
		t.Fatalf("tier 2 must be observed, got %+v", observer.calls)
	}
	if observer.calls[0].reason != aifallbackreason.Tier2ObserveOnly {
		t.Fatalf("the observation must carry the tier-2 code, got %s", observer.calls[0].reason)
	}
}

func TestTier2WhenNoPolicyCodeConfigured(t *testing.T) {
	repo := &fakeCostRepo{policy: policyWith(aioperation.Chat)}
	resolver := newResolver(repo, "", nil)

	got, err := resolver.Resolve(context.Background(), testUser, aioperation.Chat)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.Tier2 {
		t.Fatal("without a code there is nothing to look up, so tier 2 applies")
	}
	if repo.lookups != 0 {
		t.Fatal("an empty code must not trigger a pointless database query")
	}
}

func TestTier2WhenDefaultPolicyIsInactive(t *testing.T) {
	policy := policyWith(aioperation.Chat)
	policy.IsActive = false
	resolver := newResolver(&fakeCostRepo{policy: policy}, "free-default", nil)

	got, _ := resolver.Resolve(context.Background(), testUser, aioperation.Chat)
	if !got.Tier2 {
		t.Fatal("an inactive default policy must not be used")
	}
}

func TestTier2WhenOperationIsMissingFromDefaultPolicy(t *testing.T) {
	// The policy exists and is active but does not cover this operation.
	repo := &fakeCostRepo{policy: policyWith(aioperation.Summary)}
	resolver := newResolver(repo, "free-default", nil)

	got, _ := resolver.Resolve(context.Background(), testUser, aioperation.Chat)
	if !got.Tier2 {
		t.Fatal("an operation the default policy does not configure must degrade")
	}
}

func TestTier2WhenOperationIsDisabled(t *testing.T) {
	policy := policyWith(aioperation.Chat)
	policy.Operations[0].IsEnabled = false
	resolver := newResolver(&fakeCostRepo{policy: policy}, "free-default", nil)

	got, _ := resolver.Resolve(context.Background(), testUser, aioperation.Chat)
	if !got.Tier2 {
		t.Fatal("a disabled operation must not be routed to a paid model")
	}
}

func TestTier2WhenLookupFails(t *testing.T) {
	repo := &fakeCostRepo{lookupErr: errors.New("connection reset")}
	resolver := newResolver(repo, "free-default", nil)

	got, err := resolver.Resolve(context.Background(), testUser, aioperation.Chat)
	if err != nil {
		t.Fatalf("a transient database failure must degrade, not fail: %v", err)
	}
	if !got.Tier2 {
		t.Fatal("an unreadable policy must degrade to the free route")
	}
}

// TestTier1IsNotReEntered is the ordering guarantee: once tier 1 succeeds, tier 2
// is never evaluated, so a working configuration can never be reported as
// degraded.
func TestTier1IsNotReEntered(t *testing.T) {
	repo := &fakeCostRepo{policy: policyWith(aioperation.Chat)}
	observer := &recordingObserver{}
	resolver := newResolver(repo, "free-default", observer)

	if _, err := resolver.Resolve(context.Background(), testUser, aioperation.Chat); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.lookups != 1 {
		t.Fatalf("the default policy must be looked up exactly once, got %d", repo.lookups)
	}
	for _, call := range observer.calls {
		if call.tier2 {
			t.Fatal("tier 2 must not be reported when tier 1 succeeded")
		}
	}
}

// TestTier2TokenCeilingsArePositive guards the degraded prompt budget: a zero
// ceiling would silently truncate every degraded response to nothing.
func TestTier2TokenCeilingsArePositive(t *testing.T) {
	if Tier2MaxInputTokens <= 0 || Tier2MaxOutputTokens <= 0 {
		t.Fatalf("tier-2 token ceilings must be positive, got %d/%d",
			Tier2MaxInputTokens, Tier2MaxOutputTokens)
	}
}

// TestNilObserverDoesNotPanic covers the case where the counter has not been
// attached yet: losing the metric must never take down an AI request.
func TestNilObserverDoesNotPanic(t *testing.T) {
	repo := &fakeCostRepo{policy: nil}
	resolver := NewGrantFallbackResolver(repo, testConfig("missing"), logger.New("test", "", "info", false), nil)
	if _, err := resolver.Resolve(context.Background(), testUser, aioperation.Chat); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestFallbackMetricsCountsByReason covers FR-010: the counters have to separate
// the reason codes, otherwise "the default policy name is wrong" and "the
// database was briefly unreachable" look like one number and an operator
// cannot tell which to fix.
func TestFallbackMetricsCountsByReason(t *testing.T) {
	metrics := NewFallbackMetrics(logger.New("test", "", "info", false))
	ctx := context.Background()

	metrics.OnFallback(ctx, testUser, aioperation.Chat, aifallbackreason.Tier1DefaultPolicy, false)
	metrics.OnFallback(ctx, testUser, aioperation.Outfit, aifallbackreason.Tier1DefaultPolicy, false)
	metrics.OnFallback(ctx, testUser, aioperation.Chat, aifallbackreason.DefaultPolicyNotFound, true)
	metrics.OnFallback(ctx, testUser, aioperation.Chat, aifallbackreason.PolicyLookupFailed, true)

	snapshot := metrics.Snapshot()
	if snapshot[string(aifallbackreason.Tier1DefaultPolicy)] != 2 {
		t.Fatalf("tier 1 count wrong: %+v", snapshot)
	}
	if snapshot[string(aifallbackreason.DefaultPolicyNotFound)] != 1 {
		t.Fatalf("missing-policy count wrong: %+v", snapshot)
	}
	if snapshot[string(aifallbackreason.PolicyLookupFailed)] != 1 {
		t.Fatalf("lookup-failure count wrong: %+v", snapshot)
	}
	if snapshot["tier2_total"] != 2 {
		t.Fatalf("tier-2 total wrong: %+v", snapshot)
	}
}

// TestTier2WarningThrottling pins the log-volume behaviour: warnings fire on the
// first occurrence and on every doubling, not on every request.
func TestTier2WarningThrottling(t *testing.T) {
	for total, wantWarn := range map[int64]bool{
		1: true, 2: true, 3: false, 4: true, 5: false, 6: false, 7: false, 8: true, 9: false,
	} {
		if got := isWarningStep(total); got != wantWarn {
			t.Fatalf("isWarningStep(%d) = %v, want %v", total, got, wantWarn)
		}
	}
}
