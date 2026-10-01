package grant

import (
	"context"

	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/aioperation"
	aiexecutor_entities "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/entities"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/repositories"
	"smart-wardrobe-be/shared/config"
	"smart-wardrobe-be/shared/domain/constants/aifallbackreason"
	"smart-wardrobe-be/shared/pkg/logger"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Tier2 token ceilings.
//
// Tier 2 has no policy row, so there is no configured limit to read. These
// numbers match the smallest limits the free policy uses, which keeps the
// degraded response in the same shape the caller already handles instead of
// introducing a third set of magic values in each use case.
const (
	Tier2MaxInputTokens  = 3000
	Tier2MaxOutputTokens = 500
)

// FallbackResolution is the outcome of resolving the cost policy for a request
// that has no usable policy grant.
type FallbackResolution struct {
	// Policy and Operation are the tier-1 pair. Both are nil when Tier2 is true.
	Policy    *aiexecutor_entities.AICostPolicy
	Operation *aiexecutor_entities.AICostPolicyOperation

	// Tier2 is true only when tier 1 could not produce a usable policy, in which
	// case the caller must run the request on the free model.
	Tier2  bool
	Reason aifallbackreason.Reason
}

// GrantFallbackResolver replaces the hard error that used to be returned when an
// account had no policy grant row (024 FR-006).
//
// The bug it fixes is not cosmetic. An account whose grant had not arrived yet
// — brand new, or mid-delivery after a plan change — got a 500 from every AI
// call, at exactly the moment the user had just paid. Hard failure here is worse
// than degraded service, because degraded service still serves the request.
type GrantFallbackResolver struct {
	repo     repositories.IAICostRepository
	cfg      *config.Config
	logger   logger.Interface
	observed FallbackObserver
}

// FallbackObserver receives one callback per degradation so a counter can be
// advanced and a warning raised (FR-008, FR-010). It is an interface rather
// than a direct metric call so the resolver stays free of any metrics package
// and remains trivially testable.
type FallbackObserver interface {
	OnFallback(ctx context.Context, userID uuid.UUID, operation aioperation.AIOperation, reason aifallbackreason.Reason, tier2 bool)
}

func NewGrantFallbackResolver(
	repo repositories.IAICostRepository,
	cfg *config.Config,
	log logger.Interface,
	observer FallbackObserver,
) *GrantFallbackResolver {
	return &GrantFallbackResolver{repo: repo, cfg: cfg, logger: log, observed: observer}
}

// Resolve applies the two-tier degradation for an account with no policy grant.
//
// Tier 1 — use the configured default cost policy, going through the ordinary
// reservation-first ledger with a nil grant. The ledger is NOT skipped: the
// default policy routes chat and outfit to a paid model, so falling back without
// reserving would be unbounded spend with no control (research.md R8).
//
// Tier 2 — no policy at all. The caller runs the request on the free model with
// no cap and zero paid attempts allowed, and a warning is raised. This is the
// last resort, not a normal path.
func (r *GrantFallbackResolver) Resolve(ctx context.Context, userID uuid.UUID, operation aioperation.AIOperation) (*FallbackResolution, error) {
	resolution := r.resolveTier1(ctx, userID, operation)
	if resolution.Policy != nil && resolution.Operation != nil {
		r.observe(ctx, userID, operation, aifallbackreason.Tier1DefaultPolicy, false)
		return resolution, nil
	}

	r.logger.Warn("AI cost policy fallback tier 1 unusable, degrading to free model",
		zap.String("user_id", userID.String()),
		zap.String("operation", string(operation)),
		zap.String("reason", string(resolution.Reason)),
		zap.String("configured_policy_code", r.cfg.AI.DefaultCostPolicyCode),
	)

	resolution.Tier2 = true
	resolution.Reason = aifallbackreason.Tier2ObserveOnly
	r.observe(ctx, userID, operation, aifallbackreason.Tier2ObserveOnly, true)
	return resolution, nil
}

// resolveTier1 looks up the configured default policy. It returns a resolution
// with a nil Policy and a specific reason on every failure path, so the caller
// can log why tier 1 was skipped rather than just that it was.
func (r *GrantFallbackResolver) resolveTier1(ctx context.Context, userID uuid.UUID, operation aioperation.AIOperation) *FallbackResolution {
	code := r.cfg.AI.DefaultCostPolicyCode
	if code == "" {
		// Without a code there is nothing to look up. Falling back to a
		// hardcoded policy name here would make "which policy is the default"
		// a code change, which constitution §VIII forbids.
		return &FallbackResolution{Reason: aifallbackreason.DefaultPolicyNotFound}
	}

	policy, err := r.repo.GetPolicyByCode(ctx, code)
	if err != nil {
		r.logger.Error("default AI cost policy lookup failed",
			zap.String("policy_code", code),
			zap.Error(err),
		)
		return &FallbackResolution{Reason: aifallbackreason.PolicyLookupFailed}
	}
	if policy == nil {
		return &FallbackResolution{Reason: aifallbackreason.DefaultPolicyNotFound}
	}
	if !policy.IsActive {
		return &FallbackResolution{Reason: aifallbackreason.DefaultPolicyInactive}
	}

	for _, op := range policy.Operations {
		if op.Operation == operation && op.IsEnabled {
			return &FallbackResolution{Policy: policy, Operation: op}
		}
	}
	return &FallbackResolution{Reason: aifallbackreason.OperationNotConfigured}
}

func (r *GrantFallbackResolver) observe(ctx context.Context, userID uuid.UUID, operation aioperation.AIOperation, reason aifallbackreason.Reason, tier2 bool) {
	if r.observed == nil {
		return
	}
	r.observed.OnFallback(ctx, userID, operation, reason, tier2)
}

// 024 T178: `SetObserver` is GONE, and this is the note explaining why.
//
// Its doc comment claimed wire could not inject the observer through the constructor
// and that the method therefore existed for that purpose. Wire DOES inject it there —
// `NewGrantFallbackResolver` takes the observer as a parameter and assigns it to
// `observed` — so nothing ever called this. A setter for an already-injected
// dependency is not merely unused: it advertises a second, supported way to configure
// the resolver, and the next caller to use it would be quietly overriding the wiring.
//
// The comment also described the observer as a gauge from a time-series system, which
// does not exist in this repository. `FallbackObserver` is a plain interface whose
// implementation counts by reason and exposes a `Snapshot()` read by the health
// response.
