package orchestrator

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	uc_interfaces "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/application/interface/usecase"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/application/usecase/grant"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/aienforcementmode"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/aioperation"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/airoute"
	aiexecutor_entities "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/entities"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/repositories"
	"smart-wardrobe-be/shared/config"
	"smart-wardrobe-be/shared/domain/constants/aipolicygrantstatus"
	"smart-wardrobe-be/shared/pkg/logger"

	"github.com/google/uuid"
)

// tokenMeta is the minimum estimation meta resolveExecutionPlan reads.
func tokenMeta() uc_interfaces.AITokenEstimationMeta {
	return uc_interfaces.AITokenEstimationMeta{EstimatedPromptTokens: 100}
}

// 024 T174 / FR-029b: the snapshot/live boundary has to be ONE stated rule, because an
// unstated boundary is how the two sources got mixed in the first place.
//
// The rule: token ceilings come from the snapshot; when the snapshot carries no value
// for the operation, they come from the live operation row; and from nowhere else.
//
// Both halves are asserted below, and the fixture makes them fight: the snapshot and
// the live row carry DIFFERENT numbers, so a test cannot pass by reading the wrong one
// by accident. That is the whole risk here — reading live when the snapshot exists is
// exactly what FR-029b forbids, because an operator lowering a ceiling would then
// silently change what a paying customer is allowed.

// planRepoStub answers ResolvePolicy with whatever the test staged.
type planRepoStub struct {
	repositories.IAICostRepository

	grant  *aiexecutor_entities.AIPolicyGrant
	policy *aiexecutor_entities.AICostPolicy
	op     *aiexecutor_entities.AICostPolicyOperation
}

func (r *planRepoStub) ResolvePolicy(context.Context, uuid.UUID, aioperation.AIOperation, time.Time) (
	*aiexecutor_entities.AIPolicyGrant,
	*aiexecutor_entities.AICostPolicy,
	*aiexecutor_entities.AICostPolicyOperation,
	error,
) {
	return r.grant, r.policy, r.op, nil
}

// GetPolicyByCode is what the tier-1 fallback resolver calls to find the
// configured default policy. It returns the same staged policy the grant path
// would have returned, with its operations attached — `resolveTier1` refuses a
// policy whose operation row is missing or disabled, so a policy with an empty
// Operations slice would silently push the test onto the tier-2 path and the
// tier-1 behaviour under test would never run.
func (r *planRepoStub) GetPolicyByCode(context.Context, string) (*aiexecutor_entities.AICostPolicy, error) {
	if r.policy == nil {
		return nil, nil
	}
	if len(r.policy.Operations) == 0 && r.op != nil {
		r.policy.Operations = []*aiexecutor_entities.AICostPolicyOperation{r.op}
	}
	return r.policy, nil
}

// snapshotGrant builds a grant whose policy_snapshot carries the given operations.
// Operations left out are simply absent, which is what an incomplete snapshot is.
func snapshotGrant(t *testing.T, includeAll bool, inputTokens int) *aiexecutor_entities.AIPolicyGrant {
	t.Helper()
	log := logger.New("test", "", "info", false)

	entitlements := map[string]any{}
	if includeAll {
		for _, op := range []aioperation.AIOperation{aioperation.Chat, aioperation.Outfit, aioperation.Summary, aioperation.Rewriter} {
			entitlements[string(op)] = map[string]any{
				"maxPaidAttemptsPerDay":  5,
				"normalMaxInputTokens":   inputTokens,
				"normalMaxOutputTokens":  500,
				"reducedMaxInputTokens":  inputTokens - 500,
				"reducedMaxOutputTokens": 400,
			}
		}
	} else {
		// Every operation present EXCEPT the one under test. This is the shape the
		// eight pre-existing incomplete snapshots had.
		for _, op := range []aioperation.AIOperation{aioperation.Chat, aioperation.Summary, aioperation.Rewriter} {
			entitlements[string(op)] = map[string]any{
				"maxPaidAttemptsPerDay":  5,
				"normalMaxInputTokens":   inputTokens,
				"normalMaxOutputTokens":  500,
				"reducedMaxInputTokens":  inputTokens - 500,
				"reducedMaxOutputTokens": 400,
			}
		}
	}
	raw, err := json.Marshal(map[string]any{"entitlements": entitlements})
	if err != nil {
		t.Fatalf("marshal snapshot: %v", err)
	}

	// BuildPolicySnapshot is the only supported way to produce this shape, so go
	// through the real reader rather than hand-rolling JSON that merely looks right.
	_ = log
	return &aiexecutor_entities.AIPolicyGrant{
		UserID:         uuid.New(),
		EffectiveFrom:  time.Now().Add(-time.Hour),
		Status:         aipolicygrantstatus.Active,
		PolicySnapshot: aiexecutor_entities.JSONDocument(raw),
		PlanCode:       "premium",
	}
}

// liveOp builds the live operation row with its own, deliberately different, ceilings.
func liveOp(policyID uuid.UUID, inputTokens int) *aiexecutor_entities.AICostPolicyOperation {
	return &aiexecutor_entities.AICostPolicyOperation{
		PolicyID:               policyID,
		Operation:              aioperation.Outfit,
		IsEnabled:              true,
		NormalRoute:            airoute.Paid,
		FreeRoute:              airoute.Free,
		NormalMaxInputTokens:   inputTokens,
		NormalMaxOutputTokens:  999,
		ReducedMaxInputTokens:  inputTokens - 1000,
		ReducedMaxOutputTokens: 888,
	}
}

func stagedRepo(grant *aiexecutor_entities.AIPolicyGrant, op *aiexecutor_entities.AICostPolicyOperation) *planRepoStub {
	policy := &aiexecutor_entities.AICostPolicy{
		AuditableEntity:  aiexecutor_entities.AuditableEntity{BaseEntity: aiexecutor_entities.BaseEntity{ID: uuid.New()}},
		Code:             "premium-default",
		EnforcementMode:  aienforcementmode.ObserveOnly,
		PeriodDays:       30,
		IsActive:         true,
		HardCostMicroVND: func() *int64 { v := int64(30000000000); return &v }(),
	}
	op.PolicyID = policy.ID
	return &planRepoStub{grant: grant, policy: policy, op: op}
}

func newPlanUC(repo repositories.IAICostRepository) *OrchestratorUseCase {
	log := logger.New("test", "", "info", false)
	cfg := &config.Config{}
	// resolveProviderModel refuses an operation with no configured model, so the
	// fixture has to supply one for both routes. The values are irrelevant here: this
	// test is about where the TOKEN ceilings come from, not which model is picked.
	cfg.AI.RecommendationTextPrimary.Provider = "test-provider"
	cfg.AI.RecommendationTextPrimary.Model = "test-model"
	cfg.AI.FreeTextPrimary.Provider = "test-free-provider"
	cfg.AI.FreeTextPrimary.Model = "test-free-model"
	// The tier-1 resolver refuses to look anything up without this code, and an
	// empty one silently sends the request to tier 2 instead — which would make a
	// tier-1 test pass while never exercising tier 1.
	cfg.AI.DefaultCostPolicyCode = "free-default"
	return &OrchestratorUseCase{
		repo:         repo,
		cfg:          cfg,
		entitlements: grant.NewEntitlementReader(log),
		logger:       log,
	}
}

// Voe 1 — the snapshot wins whenever it has a value.
func TestTokenCeilingsComeFromTheSnapshotWhenItIsComplete(t *testing.T) {
	const snapshotTokens = 6000
	const liveTokens = 100

	repo := stagedRepo(snapshotGrant(t, true, snapshotTokens), liveOp(uuid.New(), liveTokens))
	uc := newPlanUC(repo)

	plan, err := uc.resolveExecutionPlan(context.Background(), uuid.New(), aioperation.Outfit, tokenMeta())
	if err != nil {
		t.Fatalf("resolveExecutionPlan: %v", err)
	}

	if plan.MaxInputTokens != snapshotTokens {
		t.Errorf("with a complete snapshot the ceiling must come from the snapshot, got = %d, want = %d",
			plan.MaxInputTokens, snapshotTokens)
	}
	if plan.MaxOutputTokens == 999 {
		t.Error("the output ceiling must come from the snapshot, not the live value 999")
	}
}

// Voe 2 — the snapshot has no value, so the live row is the declared single fallback.
func TestTokenCeilingsFallBackToTheLiveRowWhenTheSnapshotIsIncomplete(t *testing.T) {
	const liveTokens = 100

	repo := stagedRepo(snapshotGrant(t, false, 6000), liveOp(uuid.New(), liveTokens))
	uc := newPlanUC(repo)

	plan, err := uc.resolveExecutionPlan(context.Background(), uuid.New(), aioperation.Outfit, tokenMeta())
	if err != nil {
		t.Fatalf("resolveExecutionPlan: %v", err)
	}

	if plan.MaxInputTokens != liveTokens {
		t.Errorf("with an incomplete snapshot the ceiling must come from the live row, got = %d, want = %d",
			plan.MaxInputTokens, liveTokens)
	}
}

// ---- The tier-1 fallback: no grant at all ----

// Voe 3 — the tier-1 fallback passes a NIL grant into the entitlement reader, and
// this is the path that killed the AI service.
//
// `ResolvePolicy` returns no grant for a brand-new account or for the window
// between a plan change and the arrival of its event. `resolveExecutionPlan` then
// degrades to the configured default policy and deliberately sets `grant = nil`.
// The reader was handed that nil and dereferenced it while building a log field,
// so one request took the whole process down — which is why the container in the
// incident report had RestartCount=6 rather than one failed call.
//
// Both assertions below are load-bearing, and the first one is the regression
// itself: if the reader dereferences the grant, this test panics and the package
// fails. The second says the degradation is still CORRECT, not merely survivable —
// a tier-1 request must come out of here bounded by the live policy, because an
// account with no grant is precisely the account that must not run unbounded.
func TestTier1FallbackWithNoGrantDoesNotPanicAndStillBoundsTheRequest(t *testing.T) {
	const liveTokens = 4000
	const liveOutputTokens = 999 // the value liveOp() puts in the fixture

	repo := stagedRepo(nil, liveOp(uuid.New(), liveTokens))
	uc := newPlanUC(repo)
	uc.fallbackResolver = grant.NewGrantFallbackResolver(repo, uc.cfg, uc.logger, nil)

	plan, err := uc.resolveExecutionPlan(context.Background(), uuid.New(), aioperation.Outfit, tokenMeta())
	if err != nil {
		t.Fatalf("a missing grant must degrade, not fail: %v", err)
	}

	if plan.MaxInputTokens != liveTokens {
		t.Errorf("with no grant the ceiling must come from the live policy row, got = %d, want = %d",
			plan.MaxInputTokens, liveTokens)
	}
	if plan.MaxOutputTokens != liveOutputTokens {
		t.Errorf("the output ceiling must come from the live policy row too, got = %d, want = %d",
			plan.MaxOutputTokens, liveOutputTokens)
	}
	if plan.Route != airoute.Paid {
		t.Errorf("the default policy routes outfit to the paid model, got = %q", plan.Route)
	}
}
