package grant

import (
	"encoding/json"
	"time"

	aierrors "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/application/errors"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/aioperation"
	aiexecutor_entities "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/entities"
	"smart-wardrobe-be/shared/pkg/logger"

	"go.uber.org/zap"
)

// entitlementOperations is the closed set of operations every snapshot must
// carry. A snapshot missing any of them cannot satisfy FR-029b, because the
// reader has no limit to apply for that operation and would have to invent one.
var entitlementOperations = []aioperation.AIOperation{
	aioperation.Chat,
	aioperation.Outfit,
	aioperation.Summary,
	aioperation.Rewriter,
}

// PolicySnapshot is the two-part document written into
// ai_policy_grants.policy_snapshot (FR-029).
type PolicySnapshot struct {
	// Envelope records which event produced the grant. It is the trace back to
	// the source and the input to the ordering check that stops a late event
	// from resurrecting a revoked entitlement.
	Envelope     SnapshotEnvelope                `json:"envelope"`
	Entitlements map[string]OperationEntitlement `json:"entitlements"`
}

// SnapshotEnvelope identifies the event that produced the grant.
type SnapshotEnvelope struct {
	EventVersion  int        `json:"eventVersion"`
	EventType     string     `json:"eventType"`
	OccurredAt    time.Time  `json:"occurredAt"`
	EffectiveAt   time.Time  `json:"effectiveAt"`
	EffectiveTo   *time.Time `json:"effectiveTo,omitempty"`
	DurationDays  int        `json:"durationDays"`
	SourceEventID string     `json:"sourceEventId"`
}

// OperationEntitlement holds ONLY the limits a customer bought. Routing and
// anti-abuse settings are deliberately absent: they belong to the live
// configuration, not to what was sold.
type OperationEntitlement struct {
	MaxPaidAttemptsPerDay  *int `json:"maxPaidAttemptsPerDay"`
	NormalMaxInputTokens   int  `json:"normalMaxInputTokens"`
	NormalMaxOutputTokens  int  `json:"normalMaxOutputTokens"`
	ReducedMaxInputTokens  int  `json:"reducedMaxInputTokens"`
	ReducedMaxOutputTokens int  `json:"reducedMaxOutputTokens"`
}

// BuildPolicySnapshot assembles the snapshot from the event and the live policy.
//
// The split between the two halves is the single most consequential decision in
// this feature, and it is deliberate (research.md R13):
//
//   - ENTITLEMENT limits (per-day attempts, token ceilings) are frozen, because
//     they are what the customer bought. Reading them live would silently take
//     away something already paid for.
//   - COST-CONTROL settings (hard cap, enforcement mode, routes, anti-abuse
//     knobs) are read live, because they are the operator's brake. Freezing them
//     would make a cost cap impossible to tighten — and five free-plan accounts
//     hold grants with effective_to IS NULL that are only ever recreated when
//     the account changes plan, so a frozen cap could be lowered but never
//     actually take effect on them.
type BuildPolicySnapshotInput struct {
	Event      *PlanChangedEvent
	Policy     *aiexecutor_entities.AICostPolicy
	OccurredAt time.Time
}

// BuildPolicySnapshot is a function so the two halves can be tested separately.
func BuildPolicySnapshot(in BuildPolicySnapshotInput) (*PolicySnapshot, error) {
	if in.Policy == nil {
		// 024 T187: internal fault — see the classification note in the errors
		// package. The caller already refuses a nil policy with its own classified
		// reason before reaching here, so this is a defensive guard, not a
		// client-facing rejection.
		return nil, aierrors.ErrCostPolicyUnavailable()
	}

	entitlements := make(map[string]OperationEntitlement, len(in.Policy.Operations))
	for _, op := range in.Policy.Operations {
		entitlements[string(op.Operation)] = OperationEntitlement{
			MaxPaidAttemptsPerDay:  op.MaxPaidAttemptsPerDay,
			NormalMaxInputTokens:   op.NormalMaxInputTokens,
			NormalMaxOutputTokens:  op.NormalMaxOutputTokens,
			ReducedMaxInputTokens:  op.ReducedMaxInputTokens,
			ReducedMaxOutputTokens: op.ReducedMaxOutputTokens,
		}
	}

	return &PolicySnapshot{
		Envelope: SnapshotEnvelope{
			EventVersion:  in.Event.EventVersion,
			EventType:     string(in.Event.EventType),
			OccurredAt:    in.Event.OccurredAt,
			EffectiveAt:   in.Event.EffectiveAt,
			EffectiveTo:   in.Event.EffectiveTo,
			DurationDays:  in.Event.DurationDays,
			SourceEventID: in.Event.SourceEventID.String(),
		},
		Entitlements: entitlements,
	}, nil
}

// Marshal renders the snapshot for the jsonb column.
func (s *PolicySnapshot) Marshal() (aiexecutor_entities.JSONDocument, error) {
	raw, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}
	return aiexecutor_entities.JSONDocument(raw), nil
}

// EntitlementReader applies the snapshot/live split at the point of use.
type EntitlementReader struct {
	logger logger.Interface
}

func NewEntitlementReader(log logger.Interface) *EntitlementReader {
	return &EntitlementReader{logger: log}
}

// ResolvedLimits is the merged result used by the read path.
type ResolvedLimits struct {
	// FromSnapshot: what the customer bought.
	MaxPaidAttemptsPerDay  *int
	NormalMaxInputTokens   int
	NormalMaxOutputTokens  int
	ReducedMaxInputTokens  int
	ReducedMaxOutputTokens int
	// FromLiveConfig: the operator's brake, always current.
	HardCostMicroVND             *int64
	EnforcementMode              string
	FreeRouteThresholdBPS        int
	MaxUnknownPaidRequestsPerDay int
	UnknownHoldMinutes           int
	PeriodDays                   int
	NormalRoute                  string
	ReducedRoute                 string
	FreeRoute                    string
	// SnapshotComplete reports whether the snapshot carried all four operations.
	// False means the entitlements are unusable and the caller must degrade.
	SnapshotComplete bool
}

// 024 T165: this struct deliberately carries NO policy-is-active and NO
// per-operation-enabled flag, though both were here and both were assigned.
//
// They are not missing by oversight. Every caller obtains its policy either from
// ResolvePolicy or from the fallback resolver, and both already refuse a deactivated
// policy and a disabled operation — that is how the account reaches this struct in
// the first place. So the two values could only ever be true, and a field that cannot
// be false is worse than no field: it reads as a live input and invites the next
// person to branch on it, which would be a branch that can never run.
//
// FR-029b is satisfied by that refusal, not by a copy of the flag here: the live flag
// is read for enforcement, in the one place that decides whether a grant applies at
// all.

// Read merges the two sources for one operation.
//
// An incomplete snapshot is reported rather than silently patched. The eight
// snapshots that existed before this feature were all incomplete, and treating
// a missing limit as zero would have frozen the free policy's anti-abuse counter
// at zero — a state that looks valid and enforces nothing.
func (r *EntitlementReader) Read(
	grant *aiexecutor_entities.AIPolicyGrant,
	policy *aiexecutor_entities.AICostPolicy,
	operation aioperation.AIOperation,
) *ResolvedLimits {
	resolved := &ResolvedLimits{}
	if policy != nil {
		resolved.HardCostMicroVND = policy.HardCostMicroVND
		resolved.EnforcementMode = string(policy.EnforcementMode)
		resolved.FreeRouteThresholdBPS = policy.FreeRouteThresholdBPS
		resolved.MaxUnknownPaidRequestsPerDay = policy.MaxUnknownPaidRequestsPerDay
		resolved.UnknownHoldMinutes = policy.UnknownHoldMinutes
		resolved.PeriodDays = policy.PeriodDays
		for _, op := range policy.Operations {
			if op.Operation != operation {
				continue
			}
			resolved.NormalRoute = string(op.NormalRoute)
			resolved.ReducedRoute = string(op.ReducedRoute)
			resolved.FreeRoute = string(op.FreeRoute)
			break
		}
	}

	entitlements, err := decodeEntitlements(grant)
	if err != nil {
		r.logger.Warn("policy snapshot could not be decoded; entitlements unavailable",
			zap.String("user_id", grant.UserID.String()),
			zap.String("operation", string(operation)),
			zap.Error(err),
		)
		return resolved
	}

	missing := make([]string, 0, len(entitlementOperations))
	for _, required := range entitlementOperations {
		if _, ok := entitlements[string(required)]; !ok {
			missing = append(missing, string(required))
		}
	}
	if len(missing) > 0 {
		r.logger.Warn("policy snapshot is missing operations; entitlements treated as unavailable",
			zap.String("user_id", grant.UserID.String()),
			zap.Strings("missing_operations", missing),
		)
		return resolved
	}
	resolved.SnapshotComplete = true

	entitlement := entitlements[string(operation)]
	resolved.MaxPaidAttemptsPerDay = entitlement.MaxPaidAttemptsPerDay
	resolved.NormalMaxInputTokens = entitlement.NormalMaxInputTokens
	resolved.NormalMaxOutputTokens = entitlement.NormalMaxOutputTokens
	resolved.ReducedMaxInputTokens = entitlement.ReducedMaxInputTokens
	resolved.ReducedMaxOutputTokens = entitlement.ReducedMaxOutputTokens
	return resolved
}

func decodeEntitlements(grant *aiexecutor_entities.AIPolicyGrant) (map[string]OperationEntitlement, error) {
	// 024 T187: both failures below are corrupt stored data — the snapshot column
	// was written by this service — so they share one internal factory rather than
	// two inline errors. Two different texts used to be logged for the same class
	// of fault, which made a grep of the logs look like two separate problems.
	if grant == nil || len(grant.PolicySnapshot) == 0 {
		return nil, aierrors.ErrPolicySnapshotUnreadable()
	}
	var snapshot PolicySnapshot
	if err := json.Unmarshal(grant.PolicySnapshot, &snapshot); err != nil {
		return nil, err
	}
	if snapshot.Entitlements == nil {
		return nil, aierrors.ErrPolicySnapshotUnreadable()
	}
	return snapshot.Entitlements, nil
}

// ExtractOccurredAt returns the occurredAt recorded in the snapshot envelope.
// It is the input to the ordering check: a late event whose occurredAt is not
// newer than the stored one must be refused, otherwise an out-of-order renewal
// would revive an entitlement that was already revoked.
func ExtractOccurredAt(grant *aiexecutor_entities.AIPolicyGrant) (time.Time, bool) {
	if grant == nil || len(grant.PolicySnapshot) == 0 {
		return time.Time{}, false
	}
	var snapshot PolicySnapshot
	if err := json.Unmarshal(grant.PolicySnapshot, &snapshot); err != nil {
		return time.Time{}, false
	}
	if snapshot.Envelope.OccurredAt.IsZero() {
		return time.Time{}, false
	}
	return snapshot.Envelope.OccurredAt, true
}
