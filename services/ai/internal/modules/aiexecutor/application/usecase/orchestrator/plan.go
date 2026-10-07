package orchestrator

import (
	"context"
	"time"

	aierrors "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/application/errors"
	uc_interfaces "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/application/interface/usecase"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/application/usecase/grant"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/aienforcementmode"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/aioperation"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/airoute"
	aiexecutor_entities "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/entities"
	"smart-wardrobe-be/shared/domain/constants/aifallbackreason"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ----- Execution Plan -----

func (uc *OrchestratorUseCase) resolveExecutionPlan(ctx context.Context, userID uuid.UUID, operation aioperation.AIOperation, meta uc_interfaces.AITokenEstimationMeta) (*uc_interfaces.ExecutionPlan, error) {
	provider, model := uc.resolveProviderModel(operation, airoute.Paid)
	if model == "" {
		// 024 T187: the caller named an operation this service does not implement.
		// That is a bad request, not a server fault, so it is a factory rather than
		// an inline fmt.Errorf — see the classification note in the errors package.
		return nil, aierrors.ErrUnsupportedAIOperation(string(operation))
	}

	plan := &uc_interfaces.ExecutionPlan{
		RequestID:             uuid.New(),
		UserID:                userID,
		Operation:             operation,
		Provider:              provider,
		Model:                 model,
		Route:                 airoute.Free,
		EstimatedPromptTokens: meta.EstimatedPromptTokens,
	}

	if aioperation.IsAlwaysFree(operation) {
		return plan, nil
	}

	grant, policy, op, err := uc.repo.ResolvePolicy(ctx, userID, operation, time.Now())
	if err != nil || grant == nil {
		// The account has no usable policy grant. This used to be a hard error,
		// which meant every AI call failed for a brand-new account or for the
		// window between a plan change and the arrival of its event — exactly
		// when the user has just paid (024 FR-006). Degrade instead.
		fallback, tier2 := uc.resolveFallback(ctx, userID, operation)
		if tier2 {
			plan.Route = airoute.Free
			plan.MaxInputTokens = fallback.Operation.ReducedMaxInputTokens
			plan.MaxOutputTokens = fallback.Operation.ReducedMaxOutputTokens
			plan.Provider, plan.Model = uc.resolveProviderModel(operation, airoute.Free)
			return plan, nil
		}
		policy, op = fallback.Policy, fallback.Operation
		// `grant = nil` here is CORRECT and must stay. The account has no grant
		// row at all, so there is no purchased entitlement to read a snapshot from,
		// and inventing one would be exactly the thing FR-029b forbids. It is also
		// why `Read` below is given a nil grant on purpose rather than by accident:
		// it returns SnapshotComplete false, and the `!limits.SnapshotComplete`
		// branch below then fills the ceilings from the live operation row, which is
		// the declared single fallback for "the snapshot carries nothing for this
		// operation".
		grant = nil
	}

	// 024 T065 / T174: the token ceilings come from the grant snapshot — the limit the
	// customer actually bought — while the routes and the enforcement mode come
	// from the live policy. Reading the live per-day attempt cap here instead
	// would silently change what a paying customer is allowed.
	limits := uc.entitlements.Read(grant, policy, operation)
	if !limits.SnapshotComplete {
		// THE SINGLE RULE for token ceilings, stated once because the boundary is
		// what FR-029b is about and an unstated boundary is how the two sources got
		// mixed in the first place:
		//
		//   token ceilings come from the snapshot; when the snapshot carries no value
		//   for this operation, they come from the live operation row; and from nowhere
		//   else.
		//
		// There is no third source, and no per-field blending: all four ceilings and
		// both routes below are taken from `op` together, so a plan's token budget is
		// never half purchased and half live.
		//
		// Why the live row is an acceptable fallback rather than a violation of
		// FR-029b: FR-029b forbids reading LIMITS from live configuration because it
		// silently takes away something the customer paid for. That failure needs the
		// snapshot to EXIST and disagree with the live value — an operator lowering a
		// ceiling would then change a paying customer's allowance. Here the snapshot
		// carries nothing at all, so there is no purchased allowance to take away, and
		// the live value is the same number the policy would have produced had the
		// snapshot been built. The two paths differ only in when the value was read.
		//
		// The blast radius is bounded by data, not by luck: 06-cleanup_policy_grants.sql
		// rebuilds all four operations into every grant and asserts all four keys are
		// present, so after GĐ4 there are no grants on this path. It is reachable only
		// in the migration window, and for a grant written by a path that skipped the
		// snapshot.
		//
		// The alternative — degrading to the tier-2 constants and the free route — is
		// stricter, and was considered and rejected: it would hand a paying customer a
		// smaller allowance than their plan says during the very window when the system
		// is being migrated. The cost exposure is bounded and temporary; under-charging
		// a customer who paid is neither.
		limits.NormalMaxInputTokens = op.NormalMaxInputTokens
		limits.NormalMaxOutputTokens = op.NormalMaxOutputTokens
		limits.ReducedMaxInputTokens = op.ReducedMaxInputTokens
		limits.ReducedMaxOutputTokens = op.ReducedMaxOutputTokens
		// Routes are live-only by FR-029b regardless of the snapshot, so these two
		// assignments are not part of the fallback above.
		limits.NormalRoute = string(op.NormalRoute)
		limits.FreeRoute = string(op.FreeRoute)
	}
	plan.Route = airoute.AIRoute(limits.NormalRoute)
	plan.MaxInputTokens = limits.NormalMaxInputTokens
	plan.MaxOutputTokens = limits.NormalMaxOutputTokens

	if limits.EnforcementMode == string(aienforcementmode.FreeOnly) {
		plan.Route = airoute.AIRoute(limits.FreeRoute)
		plan.MaxInputTokens = limits.ReducedMaxInputTokens
		plan.MaxOutputTokens = limits.ReducedMaxOutputTokens

		prov, mod := uc.resolveProviderModel(operation, plan.Route)
		plan.Provider = prov
		plan.Model = mod
	}

	return plan, nil
}

// resolveFallback runs the two-tier degradation and returns the policy pair to
// use. In tier 2 the returned operation row carries the free route's token
// ceilings, so the caller can still bound the prompt instead of sending an
// unbounded one.
func (uc *OrchestratorUseCase) resolveFallback(ctx context.Context, userID uuid.UUID, operation aioperation.AIOperation) (*grant.FallbackResolution, bool) {
	resolution, err := uc.fallbackResolver.Resolve(ctx, userID, operation)
	if err != nil {
		// Resolve only returns an error for a programming fault; degrade to the
		// free route rather than propagate, keeping the invariant that this path
		// never fails a paid request.
		uc.logger.Error("fallback resolver failed",
			zap.String("user_id", userID.String()),
			zap.String("operation", string(operation)),
			zap.Error(err),
		)
		return tier2Resolution(aifallbackreason.PolicyLookupFailed), true
	}

	if resolution.Tier2 {
		return tier2Resolution(resolution.Reason), true
	}
	return resolution, false
}

// tier2Resolution wraps a tier-2 reason together with the degraded token
// budget, so both the orchestrator and the cost use case see the same numbers.
func tier2Resolution(reason aifallbackreason.Reason) *grant.FallbackResolution {
	return &grant.FallbackResolution{
		Tier2:  true,
		Reason: reason,
		Operation: &aiexecutor_entities.AICostPolicyOperation{
			ReducedMaxInputTokens:  grant.Tier2MaxInputTokens,
			ReducedMaxOutputTokens: grant.Tier2MaxOutputTokens,
		},
	}
}

func (uc *OrchestratorUseCase) resolveProviderModel(operation aioperation.AIOperation, route airoute.AIRoute) (string, string) {
	if route == airoute.Free || route == airoute.Local {
		return uc.cfg.AI.FreeTextPrimary.Provider, uc.cfg.AI.FreeTextPrimary.Model
	}
	switch operation {
	case aioperation.Chat:
		return uc.cfg.AI.ChatTextPrimary.Provider, uc.cfg.AI.ChatTextPrimary.Model
	case aioperation.Outfit:
		return uc.cfg.AI.RecommendationTextPrimary.Provider, uc.cfg.AI.RecommendationTextPrimary.Model
	case aioperation.Summary:
		return uc.cfg.AI.ChatTextPrimary.Provider, uc.cfg.AI.ChatTextPrimary.Model
	case aioperation.Rewriter:
		return uc.cfg.AI.ChatTextPrimary.Provider, uc.cfg.AI.ChatTextPrimary.Model
	case aioperation.Vision:
		return uc.cfg.AI.VisionPrimary.Provider, uc.cfg.AI.VisionPrimary.Model
	case aioperation.Embedding:
		return uc.cfg.AI.EmbeddingPrimary.Provider, uc.cfg.AI.EmbeddingPrimary.Model
	default:
		return "", ""
	}
}
