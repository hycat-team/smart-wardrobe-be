package cost

import (
	"context"
	"errors"
	"time"

	aiexecutor_dto "smart-wardrobe-be/services/ai/dto/aiexecutor"
	aierrors "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/application/errors"
	uc_interfaces "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/application/interface/usecase"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/application/usecase/grant"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/aioperation"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/airoute"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/aiusageeventstatus"
	aiexecutor_entities "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/entities"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/repositories"
	"smart-wardrobe-be/shared/application/constants/apperror"
	"smart-wardrobe-be/shared/domain/constants/aifallbackreason"

	"github.com/google/uuid"
)

// providerOpenAI is the trial provider whose calls are never charged. The
// classification is provider-based on purpose: the capability list stays
// capability-only and cannot make a Gemini text fallback free.
const providerOpenAI = "openai"

type CostUseCase struct {
	repo      repositories.IAICostRepository
	pricingUC uc_interfaces.IPricingUseCase
	// 024 T178: `userSubContract` was removed here. It was injected, stored, and never
	// read — nothing in this use case ever called the Account service's subscription
	// contract. Besides being dead, it was the AI service holding a dependency on
	// another service's contract for no reason, which is exactly the kind of coupling
	// constitution §II exists to keep out.
	fallbackResolver *grant.GrantFallbackResolver
	// 024 T065: applies the snapshot/live split. Entitlement limits come from the
	// grant snapshot, cost-control settings from the live policy.
	entitlements *grant.EntitlementReader
	// 024 T138: the observer, so a degradation recorded on THIS path is counted
	// like every other one. Without it the observe_only misconfiguration produced
	// no record at all, which is what FR-005e's mandatory warning requires.
	fallbackMetrics *grant.FallbackMetrics
}

func NewCostUseCase(
	repo repositories.IAICostRepository,
	pricingUC uc_interfaces.IPricingUseCase,
	fallbackResolver *grant.GrantFallbackResolver,
	fallbackMetrics *grant.FallbackMetrics,
	entitlements *grant.EntitlementReader,
) uc_interfaces.IAICostUseCase {
	return &CostUseCase{
		repo:             repo,
		pricingUC:        pricingUC,
		fallbackResolver: fallbackResolver,
		entitlements:     entitlements,
		fallbackMetrics:  fallbackMetrics,
	}
}

func (uc *CostUseCase) Prepare(ctx context.Context, plan *uc_interfaces.ExecutionPlan) (*uc_interfaces.AICostDecision, error) {
	now := time.Now()

	// The Ollama trial provider is unbilled. Prepare returns the business route
	// but writes nothing: the event writer for this path is AdmitUnbilledAttempt,
	// called from the attempt hook immediately before the provider call.
	if plan.Provider == providerOpenAI {
		return &uc_interfaces.AICostDecision{
			RequestID:       plan.RequestID,
			Route:           plan.Route,
			MaxInputTokens:  plan.MaxInputTokens,
			MaxOutputTokens: plan.MaxOutputTokens,
			Tracked:         true,
		}, nil
	}

	if aioperation.IsAlwaysFree(plan.Operation) {
		return uc.prepareFreeEvent(ctx, plan, 0, tier2MaxOutputTokens)
	}

	grant, policy, op, err := uc.repo.ResolvePolicy(ctx, plan.UserID, plan.Operation, now)
	if err != nil || grant == nil {
		return uc.prepareWithFallback(ctx, plan, now)
	}
	return uc.reserveAgainstPolicy(ctx, plan, now, grant, policy, op)
}

// AdmitUnbilledAttempt mirrors the existing admission decision for a provider
// attempt that has no monetary cost. It never consults pricing and never
// reserves: it only writes/updates the usage event and, on the paid route under
// observe_only, consumes the existing daily attempt cap.
//
// When the cap is exhausted the request degrades to the free route instead of
// returning a new 429; the caller still runs the same Ollama attempt. A repeated
// call is idempotent because the repository upserts by request_id.
func (uc *CostUseCase) AdmitUnbilledAttempt(ctx context.Context, admission uc_interfaces.UnbilledAttemptAdmission) (*uc_interfaces.AICostDecision, error) {
	if admission.Plan == nil {
		return nil, apperror.NewInternalError("Không đọc được kế hoạch gọi AI.")
	}
	plan := admission.Plan
	now := time.Now()

	provider := admission.Provider
	if provider == "" {
		provider = plan.Provider
	}
	model := admission.Model
	if model == "" {
		model = plan.Model
	}

	grant, policy, op := uc.resolveAttemptPolicy(ctx, plan, now)
	cap := uc.resolvePaidAttemptCap(grant, policy, op)

	event := unbilledAttemptEvent(plan, provider, model, plan.Route)
	admitted, err := uc.repo.AdmitUnbilledAttempt(ctx, policy, cap, event, now)
	if err != nil {
		return nil, err
	}
	if admitted {
		return &uc_interfaces.AICostDecision{
			RequestID:       plan.RequestID,
			Route:           plan.Route,
			MaxInputTokens:  plan.MaxInputTokens,
			MaxOutputTokens: plan.MaxOutputTokens,
			Tracked:         true,
		}, nil
	}

	// Cap exhausted: keep the pre-existing degrade behaviour. The event records
	// the free route, no reservation is made, and the provider attempt is still
	// allowed so the user gets an answer instead of a new 429.
	freeEvent := unbilledAttemptEvent(plan, provider, model, airoute.Free)
	if _, err := uc.repo.AdmitUnbilledAttempt(ctx, policy, nil, freeEvent, now); err != nil {
		return nil, err
	}
	return &uc_interfaces.AICostDecision{
		RequestID:       plan.RequestID,
		Route:           airoute.Free,
		MaxInputTokens:  plan.MaxInputTokens,
		MaxOutputTokens: plan.MaxOutputTokens,
		Tracked:         true,
	}, nil
}

// PrepareBillableAttempt upgrades the trial request's existing event to the
// billable Gemini fallback. Pricing is read and the reservation is written BEFORE
// the fallback provider is called, preserving reservation-first while keeping a
// single usage event per request.
func (uc *CostUseCase) PrepareBillableAttempt(ctx context.Context, admission uc_interfaces.BillableAttemptAdmission) (*uc_interfaces.AICostDecision, error) {
	if admission.Plan == nil {
		return nil, apperror.NewInternalError("Không đọc được kế hoạch gọi AI.")
	}
	plan := admission.Plan
	now := time.Now()

	provider := admission.Provider
	if provider == "" {
		provider = plan.Provider
	}
	model := admission.Model
	if model == "" {
		model = plan.Model
	}

	grant, policy, _ := uc.resolveAttemptPolicy(ctx, plan, now)
	if policy == nil {
		policy = &aiexecutor_entities.AICostPolicy{}
	}

	pricing, err := uc.pricingUC.GetActivePricingConfig(ctx, provider, model)
	if err != nil {
		return nil, err
	}

	reserve := microCost(plan.EstimatedPromptTokens, int64(plan.MaxOutputTokens),
		pricing.InputUSDPerMillion, pricing.OutputUSDPerMillion, pricing.USDToVND)

	unknownHold := policy.UnknownHoldMinutes
	if unknownHold <= 0 {
		unknownHold = defaultUnknownHoldMinutes
	}
	unknownExpires := now.Add(time.Duration(unknownHold) * time.Minute)

	event := &aiexecutor_entities.AIUsageEvent{
		RequestID:                plan.RequestID,
		UserID:                   userIDPointer(plan.UserID),
		Operation:                plan.Operation,
		LogicalRoute:             plan.Route,
		Provider:                 &provider,
		Model:                    &model,
		PricingConfigID:          &pricing.ID,
		InputUSDPerMillion:       &pricing.InputUSDPerMillion,
		OutputUSDPerMillion:      &pricing.OutputUSDPerMillion,
		USDToVND:                 &pricing.USDToVND,
		ReservedCostMicroVND:     reserve,
		EstimatedMaxCostMicroVND: reserve,
		Status:                   aiusageeventstatus.Reserved,
		UnknownExpiresAt:         &unknownExpires,
	}

	prepared, err := uc.repo.PrepareBillableAttempt(ctx, grant, policy, event, now)
	if err != nil {
		return nil, err
	}
	if !prepared {
		return nil, apperror.NewInternalError("Không chuẩn bị được lượt gọi AI có tính phí.")
	}

	return &uc_interfaces.AICostDecision{
		RequestID:       plan.RequestID,
		Route:           plan.Route,
		MaxInputTokens:  plan.MaxInputTokens,
		MaxOutputTokens: plan.MaxOutputTokens,
		Tracked:         true,
	}, nil
}

// resolveAttemptPolicy resolves the policy pair with the same fallback rules as
// Prepare. It returns nil values when neither a grant nor a tier-1/tier-2
// resolution is available, which an unbilled attempt treats as "no attempt cap".
func (uc *CostUseCase) resolveAttemptPolicy(ctx context.Context, plan *uc_interfaces.ExecutionPlan, now time.Time) (*aiexecutor_entities.AIPolicyGrant, *aiexecutor_entities.AICostPolicy, *aiexecutor_entities.AICostPolicyOperation) {
	grant, policy, op, err := uc.repo.ResolvePolicy(ctx, plan.UserID, plan.Operation, now)
	if err == nil && grant != nil {
		return grant, policy, op
	}
	if uc.fallbackResolver == nil {
		return nil, nil, nil
	}
	resolution, resolveErr := uc.fallbackResolver.Resolve(ctx, plan.UserID, plan.Operation)
	if resolveErr != nil || resolution.Tier2 {
		return nil, nil, nil
	}
	return nil, resolution.Policy, resolution.Operation
}

func unbilledAttemptEvent(plan *uc_interfaces.ExecutionPlan, provider, model string, route airoute.AIRoute) *aiexecutor_entities.AIUsageEvent {
	return &aiexecutor_entities.AIUsageEvent{
		RequestID:    plan.RequestID,
		UserID:       userIDPointer(plan.UserID),
		Operation:    plan.Operation,
		LogicalRoute: route,
		Provider:     &provider,
		Model:        &model,
		Status:       aiusageeventstatus.Reserved,
	}
}

func userIDPointer(userID uuid.UUID) *uuid.UUID {
	if userID == uuid.Nil {
		return nil
	}
	return &userID
}

// prepareWithFallback handles an account with no usable policy grant. It must
// never fail the request: a missing grant is a degraded-entitlement condition,
// not a server error, and returning an error here is what made brand-new
// accounts and post-payment windows fail on every AI call (024 FR-006).
func (uc *CostUseCase) prepareWithFallback(ctx context.Context, plan *uc_interfaces.ExecutionPlan, now time.Time) (*uc_interfaces.AICostDecision, error) {
	resolution, err := uc.fallbackResolver.Resolve(ctx, plan.UserID, plan.Operation)
	if err != nil || resolution.Tier2 {
		return buildTier2Decision(ctx, uc.repo, plan.RequestID, plan.UserID, plan.Operation, plan.Provider, plan.Model, plan.EstimatedPromptTokens, now)
	}

	// Tier 1 keeps the reservation-first ledger of constitution §VI, with a nil
	// grant. Skipping it is not an option: the default policy routes chat and
	// outfit to a paid model, so an unreserved fallback would be unbounded spend
	// (research.md R8).
	return uc.reserveAgainstPolicy(ctx, plan, now, nil, resolution.Policy, resolution.Operation)
}

// reserveAgainstPolicy runs the reservation step for a policy pair. grant is nil
// on the tier-1 fallback path, which is what makes the ledger row fall back to
// being keyed on the account instead of on a grant id.
func (uc *CostUseCase) reserveAgainstPolicy(
	ctx context.Context,
	plan *uc_interfaces.ExecutionPlan,
	now time.Time,
	grant *aiexecutor_entities.AIPolicyGrant,
	policy *aiexecutor_entities.AICostPolicy,
	op *aiexecutor_entities.AICostPolicyOperation,
) (*uc_interfaces.AICostDecision, error) {
	route := plan.Route
	maxIn, maxOut := plan.MaxInputTokens, plan.MaxOutputTokens

	// A free route reserves nothing, so it does not need a pricing row. Looking
	// one up anyway made a missing FREE model price a hard failure for a request
	// that costs nothing, and the free config is a separate key (or a zero-cost
	// model), not a cheaper copy of the paid one.
	var pricing *aiexecutor_dto.PricingConfigDTO
	if route == airoute.Paid {
		var err error
		pricing, err = uc.pricingUC.GetActivePricingConfig(ctx, plan.Provider, plan.Model)
		if err != nil {
			return nil, err
		}
	}

	reserve := int64(0)
	var pricingConfigID *uuid.UUID
	if pricing != nil {
		pricingConfigID = &pricing.ID
		reserve = microCost(plan.EstimatedPromptTokens, int64(maxOut), pricing.InputUSDPerMillion, pricing.OutputUSDPerMillion, pricing.USDToVND)
	}
	unknownExpires := now.Add(time.Duration(policy.UnknownHoldMinutes) * time.Minute)

	event := &aiexecutor_entities.AIUsageEvent{
		RequestID: plan.RequestID,
		LedgerID:  nil,
		UserID: func() *uuid.UUID {
			if plan.UserID == uuid.Nil {
				return nil
			}
			return &plan.UserID
		}(),
		Operation:                plan.Operation,
		LogicalRoute:             route,
		PricingConfigID:          pricingConfigID,
		Provider:                 &plan.Provider,
		Model:                    &plan.Model,
		ReservedCostMicroVND:     reserve,
		EstimatedMaxCostMicroVND: reserve,
		Status:                   aiusageeventstatus.Reserved,
		UnknownExpiresAt:         &unknownExpires,
	}
	if pricing != nil {
		event.InputUSDPerMillion = &pricing.InputUSDPerMillion
		event.OutputUSDPerMillion = &pricing.OutputUSDPerMillion
		event.USDToVND = &pricing.USDToVND
	}

	// 024 T130: the per-day cap comes from the grant SNAPSHOT when there is a
	// grant, and from the live policy only when there is none.
	//
	// FR-029b draws one line: the snapshot decides what the customer bought, the
	// live configuration decides what the system risks. Reading the live column
	// here - which is what this call site used to do, inside the repository -
	// meant an administrator editing `max_paid_attempts_per_day` changed the limit
	// for accounts that had already bought a different number, and the
	// `replayRequired` warning the admin API returns was describing something that
	// was not happening.
	//
	// With no grant at all there is no snapshot to read, so the live column is the
	// only source available. That is the tier-1 fallback, and the requirement does
	// not describe a snapshot there because none exists.
	cap := uc.resolvePaidAttemptCap(grant, policy, op)

	_, admitted, err := uc.repo.Reserve(ctx, grant, policy, op, cap, event, now)
	if err != nil {
		// A broken observe_only configuration is reported as a sentinel rather
		// than a transport failure, and it must degrade instead of failing: the
		// account's only bound is the per-day counter, so the paid route is
		// refused and the request continues on the free model (FR-005e).
		if errors.Is(err, aierrors.ErrObserveOnlyMissingAttemptCap) {
			// 024 T138: the warning FR-005e makes MANDATORY was missing, because
			// this path reaches `buildTier2Decision` directly and never passes the
			// fallback resolver - which is the only place that records a
			// degradation. The dedicated reason code existed and nothing used it.
			// Recorded here, on the one branch that can produce this error, so the
			// misconfiguration is visible with a count an operator can watch.
			uc.fallbackMetrics.OnFallback(ctx, plan.UserID, plan.Operation,
				aifallbackreason.ObserveOnlyAttemptCapMissing, true)
			return buildTier2Decision(ctx, uc.repo, plan.RequestID, plan.UserID, plan.Operation, plan.Provider, plan.Model, plan.EstimatedPromptTokens, now)
		}
		return nil, err
	}
	if !admitted {
		return uc.PrepareFree(ctx, plan.RequestID, plan.UserID, plan.Operation, 0, maxOut)
	}
	return &uc_interfaces.AICostDecision{
		RequestID:       plan.RequestID,
		Route:           route,
		MaxInputTokens:  maxIn,
		MaxOutputTokens: maxOut,
		Tracked:         true,
	}, nil
}

// prepareFreeEvent writes an untracked-cost usage event on the free route.
// Shared by the operations that never touch a paid model and by the tier-2
// degradation, both of which must leave the reservation ledger untouched.
func (uc *CostUseCase) prepareFreeEvent(ctx context.Context, plan *uc_interfaces.ExecutionPlan, promptTokens int64, maxOutputTokens int) (*uc_interfaces.AICostDecision, error) {
	now := time.Now()
	event := &aiexecutor_entities.AIUsageEvent{
		RequestID: plan.RequestID,
		UserID: func() *uuid.UUID {
			if plan.UserID == uuid.Nil {
				return nil
			}
			return &plan.UserID
		}(),
		Operation:    plan.Operation,
		LogicalRoute: airoute.Free,
		Provider:     &plan.Provider,
		Model:        &plan.Model,
		PromptTokens: promptTokens,
		Status:       aiusageeventstatus.Reserved,
		UnknownExpiresAt: func() *time.Time {
			t := now.Add(30 * time.Minute)
			return &t
		}(),
	}
	if err := uc.repo.CreateEvent(ctx, event); err != nil {
		return nil, err
	}
	return &uc_interfaces.AICostDecision{
		RequestID:       plan.RequestID,
		Route:           airoute.Free,
		MaxInputTokens:  plan.MaxInputTokens,
		MaxOutputTokens: maxOutputTokens,
		Tracked:         true,
	}, nil
}

// PrepareFree records a free-route request and returns its decision.
//
// 024 T075/T076: the Account subscription lookup is gone. The AI service is not
// allowed to read the Account service's subscription table, and the plan id it
// used to fetch was then used only to pass straight back into ResolvePolicy,
// which no longer takes one. The token ceiling now comes from the entitlement
// reader, which reads it from the grant snapshot — the limit the customer
// actually bought — and only falls back to live configuration when the snapshot
// is unreadable.
func (uc *CostUseCase) PrepareFree(ctx context.Context, requestID uuid.UUID, userID uuid.UUID, operation aioperation.AIOperation, promptTokens int64, maxOutputTokens int) (*uc_interfaces.AICostDecision, error) {
	now := time.Now()
	unknownHold := defaultUnknownHoldMinutes
	maxInput := tier2MaxInputTokens

	grant, policy, op, resolveErr := uc.repo.ResolvePolicy(ctx, userID, operation, now)
	if resolveErr == nil && grant != nil {
		limits := uc.entitlements.Read(grant, policy, operation)
		if unknownHold > 0 && policy != nil && policy.UnknownHoldMinutes > 0 {
			unknownHold = policy.UnknownHoldMinutes
		}
		if limits.SnapshotComplete && limits.NormalMaxInputTokens > 0 {
			maxInput = limits.NormalMaxInputTokens
		} else if op != nil && op.NormalMaxInputTokens > 0 {
			maxInput = op.NormalMaxInputTokens
		}
	}

	event := &aiexecutor_entities.AIUsageEvent{
		RequestID: requestID,
		UserID: func() *uuid.UUID {
			if userID == uuid.Nil {
				return nil
			}
			return &userID
		}(),
		Operation:    operation,
		LogicalRoute: airoute.Free,
		PromptTokens: promptTokens,
		Status:       aiusageeventstatus.Reserved,
		UnknownExpiresAt: func() *time.Time {
			t := now.Add(time.Duration(unknownHold) * time.Minute)
			return &t
		}(),
	}
	_ = uc.repo.CreateEvent(ctx, event)
	return &uc_interfaces.AICostDecision{
		RequestID:       requestID,
		Route:           airoute.Free,
		MaxInputTokens:  maxInput,
		MaxOutputTokens: maxOutputTokens,
		Tracked:         true,
	}, nil
}

func (uc *CostUseCase) MarkInFlight(ctx context.Context, requestID uuid.UUID) error {
	return uc.repo.MarkInFlight(ctx, requestID, time.Now())
}

// Reconciliation codes recorded on confirmed events with no billable amount.
// They are data for the admin drill-down, not error paths: every Confirm
// branch below still lands the event in status Confirmed, so error metrics
// (which only count Released/Unknown/Expired) never grow because of them.
const (
	errorCodePricingConfigMissing = "pricing_config_missing"
	errorCodeUsageUnreported      = "usage_unreported"
)

func (uc *CostUseCase) Confirm(ctx context.Context, requestID uuid.UUID, usage uc_interfaces.AIUsage) error {
	now := time.Now()
	// (0a) The winning provider is the unbilled trial provider. Guarding on the
	// provider itself (not only on usage.Unbilled) keeps the cost at zero even
	// when a caller passes a wrong flag or a fallback path forgot to stamp it.
	if usage.Provider == providerOpenAI {
		return uc.repo.Confirm(ctx, requestID, usage.PromptTokens, usage.OutputTokens, usage.ThinkingTokens, 0, usage.Provider, usage.Model, usage.FinishReason, "", now)
	}
	// (0) The request ran on an unbilled capability (FREE_TEXT_* or any other
	// key listed in AI_UNBILLED_CAPABILITIES). The team is not charged for that
	// key, so the ledger records zero cost and skips pricing entirely: a free
	// config is a separate quota, not a cheaper model price. Tokens still land
	// so the dashboard can show usage even when the cost is zero.
	if usage.Unbilled {
		return uc.repo.Confirm(ctx, requestID, usage.PromptTokens, usage.OutputTokens, usage.ThinkingTokens, 0, usage.Provider, usage.Model, usage.FinishReason, "", now)
	}
	// (4) Empty provider and model: nothing to price against. Replaces the old
	// early return, which confirmed silently with no code left to reconcile.
	if usage.Provider == "" && usage.Model == "" {
		return uc.repo.Confirm(ctx, requestID, 0, 0, 0, 0, "", "", "", errorCodePricingConfigMissing, now)
	}
	// (3) Non-empty provider/model but no active pricing row: the user keeps
	// flowing on the existing policy, so the event is still confirmed with
	// actual 0 and a code for admin to reconcile instead of blocking the call.
	pricing, err := uc.pricingUC.GetActivePricingConfig(ctx, usage.Provider, usage.Model)
	if err != nil {
		return uc.repo.Confirm(ctx, requestID, usage.PromptTokens, usage.OutputTokens, usage.ThinkingTokens, 0, usage.Provider, usage.Model, usage.FinishReason, errorCodePricingConfigMissing, now)
	}
	// (5) Priced model but no reported usage: record actual 0 rather than
	// inventing numbers the provider never reported.
	if usage.PromptTokens+usage.OutputTokens+usage.ThinkingTokens == 0 {
		return uc.repo.Confirm(ctx, requestID, usage.PromptTokens, usage.OutputTokens, usage.ThinkingTokens, 0, usage.Provider, usage.Model, usage.FinishReason, errorCodeUsageUnreported, now)
	}
	// (1) Paid stays exactly as before; (2) an always-free route that still runs
	// on a billed key (vision, and any capability not in AI_UNBILLED_CAPABILITIES)
	// flows through the same computation, so its real provider cost lands in the
	// ledger and the margin report stays truthful. Requests whose winning
	// config is unbilled reach branch (0) above instead.
	cost := microCost(usage.PromptTokens, usage.OutputTokens+usage.ThinkingTokens, pricing.InputUSDPerMillion, pricing.OutputUSDPerMillion, pricing.USDToVND)
	return uc.repo.Confirm(ctx, requestID, usage.PromptTokens, usage.OutputTokens, usage.ThinkingTokens, cost, usage.Provider, usage.Model, usage.FinishReason, "", now)
}

func (uc *CostUseCase) Release(ctx context.Context, requestID uuid.UUID, reason string) error {
	return uc.repo.Release(ctx, requestID, reason, time.Now())
}

func (uc *CostUseCase) MarkUnknown(ctx context.Context, requestID uuid.UUID, reason string) error {
	return uc.repo.MarkUnknown(ctx, requestID, reason, time.Now().Add(time.Hour))
}

func (uc *CostUseCase) ExpireUnknown(ctx context.Context, now time.Time, limit int) (int64, error) {
	return uc.repo.ExpireUnknown(ctx, now, limit)
}

func (uc *CostUseCase) GetAdminAIUsageStats(ctx context.Context, snapshotDate time.Time) (*aiexecutor_dto.AIUsageAdminStatsDTO, error) {
	return uc.repo.GetAdminAIUsageStats(ctx, snapshotDate)
}

func (uc *CostUseCase) GetAIEventsDrillDown(ctx context.Context, status string, page, limit int) ([]aiexecutor_dto.AIEventDrillDownDTO, int64, error) {
	return uc.repo.GetAIEventsDrillDown(ctx, status, page, limit)
}

func (uc *CostUseCase) GetAIUsageByOperation(ctx context.Context, fromDate, toDate time.Time) ([]aiexecutor_dto.AIUsageByOperationDTO, error) {
	return uc.repo.GetAIUsageByOperation(ctx, fromDate, toDate)
}

func (uc *CostUseCase) GetTotalAIRequestCount(ctx context.Context, fromDate, toDate time.Time) (int64, error) {
	return uc.repo.GetTotalAIRequestCount(ctx, fromDate, toDate)
}

func (uc *CostUseCase) GetAIErrorBreakdown(ctx context.Context, from, to time.Time) ([]aiexecutor_dto.AIErrorBreakdownDTO, error) {
	return uc.repo.GetAIErrorBreakdown(ctx, from, to)
}

// resolvePaidAttemptCap returns the per-day paid-attempt cap to enforce.
//
// FR-029b: with a grant, the SNAPSHOT decides - the customer bought the number
// that was frozen when their plan changed. Without a grant there is no snapshot,
// and the live policy column is the only thing available.
//
// It returns nil when neither source has a number, which the repository treats as
// a refusal to admit the paid route rather than as "unlimited". nil is therefore
// never silently permissive.
func (uc *CostUseCase) resolvePaidAttemptCap(
	grant *aiexecutor_entities.AIPolicyGrant,
	policy *aiexecutor_entities.AICostPolicy,
	op *aiexecutor_entities.AICostPolicyOperation,
) *int {
	if grant != nil && uc.entitlements != nil {
		// The reader is the single implementation of the snapshot/live split, so
		// going through it keeps this call site from becoming a second one.
		return uc.entitlements.Read(grant, policy, planOperationOf(op)).MaxPaidAttemptsPerDay
	}
	if op == nil {
		return nil
	}
	return op.MaxPaidAttemptsPerDay
}

// planOperationOf extracts the operation key from a policy operation row, or the
// zero value when the row is absent.
func planOperationOf(op *aiexecutor_entities.AICostPolicyOperation) aioperation.AIOperation {
	if op == nil {
		return ""
	}
	return op.Operation
}
