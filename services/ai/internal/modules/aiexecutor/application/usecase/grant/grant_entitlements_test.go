package grant

import (
	"encoding/json"
	"testing"

	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/aienforcementmode"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/aioperation"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/airoute"
	aiexecutor_entities "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/entities"
	"smart-wardrobe-be/shared/pkg/logger"

	"github.com/google/uuid"
)

// This file pins the one boundary that decides whether a customer keeps what
// they bought: the snapshot/live split (FR-029b, research.md R13).
//
//   - Entitlement limits (per-day attempts, token ceilings) are FROZEN. Reading
//     them live would silently take away something already paid for.
//   - Cost-control settings (hard cap, enforcement mode, anti-abuse knobs) are
//     LIVE. Freezing them would make a cap impossible to tighten — and five
//     free-plan accounts hold grants with effective_to IS NULL that are only ever
//     recreated when the account changes plan, so a frozen cap could be lowered
//     and never actually take effect on them.

func newReader() *EntitlementReader {
	return NewEntitlementReader(logger.New("test", "", "info", false))
}

func livePolicy(hardCap *int64, attemptsPerDay *int) *aiexecutor_entities.AICostPolicy {
	policy := &aiexecutor_entities.AICostPolicy{
		AuditableEntity:              aiexecutor_entities.AuditableEntity{BaseEntity: aiexecutor_entities.BaseEntity{ID: uuid.New()}},
		Code:                         "free-default",
		EnforcementMode:              aienforcementmode.ObserveOnly,
		PeriodDays:                   30,
		HardCostMicroVND:             hardCap,
		FreeRouteThresholdBPS:        9200,
		UnknownHoldMinutes:           1440,
		MaxUnknownPaidRequestsPerDay: 2,
		IsActive:                     true,
	}
	for _, op := range entitlementOperations {
		policy.Operations = append(policy.Operations, &aiexecutor_entities.AICostPolicyOperation{
			PolicyID:              policy.ID,
			Operation:             op,
			NormalRoute:           airoute.Paid,
			FreeRoute:             airoute.Free,
			MaxPaidAttemptsPerDay: attemptsPerDay,
			IsEnabled:             true,
		})
	}
	return policy
}

func grantWithEntitlements(t *testing.T, perDay *int, inputTokens int) *aiexecutor_entities.AIPolicyGrant {
	t.Helper()
	reader := newReader()
	entitlements := make(map[string]OperationEntitlement, len(entitlementOperations))
	for _, op := range entitlementOperations {
		entitlements[string(op)] = OperationEntitlement{
			MaxPaidAttemptsPerDay:  perDay,
			NormalMaxInputTokens:   inputTokens,
			NormalMaxOutputTokens:  500,
			ReducedMaxInputTokens:  inputTokens - 500,
			ReducedMaxOutputTokens: 400,
		}
	}
	document, err := json.Marshal(&PolicySnapshot{
		Envelope:     SnapshotEnvelope{EventVersion: 2, OccurredAt: windowBase},
		Entitlements: entitlements,
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	_ = reader
	return &aiexecutor_entities.AIPolicyGrant{
		AuditableEntity: aiexecutor_entities.AuditableEntity{BaseEntity: aiexecutor_entities.BaseEntity{ID: uuid.New()}},
		PolicySnapshot:  aiexecutor_entities.JSONDocument(document),
	}
}

// TestAttemptLimitIsReadFromTheSnapshotNotLive is the load-bearing assertion:
// changing the live column must NOT change what an existing account is limited
// to. That is the whole point of freezing entitlements.
func TestAttemptLimitIsReadFromTheSnapshotNotLive(t *testing.T) {
	grant := grantWithEntitlements(t, intPtr(4), 6000)
	reader := newReader()

	// The operator changes the live column from 4 to 99.
	policy := livePolicy(nil, intPtr(99))
	limits := reader.Read(grant, policy, aioperation.Outfit)

	if limits.MaxPaidAttemptsPerDay == nil || *limits.MaxPaidAttemptsPerDay != 4 {
		t.Fatalf("the frozen limit must win over the live column, got %v", limits.MaxPaidAttemptsPerDay)
	}
}

// TestHardCapIsReadFromLiveConfigNotSnapshot is the other half: the operator's
// brake must take effect immediately, on every account, including the ones whose
// grant never refreshes.
func TestHardCapIsReadFromLiveConfigNotSnapshot(t *testing.T) {
	grant := grantWithEntitlements(t, intPtr(4), 6000)
	reader := newReader()

	policy := livePolicy(int64Ptr(30000000000), intPtr(4))
	limits := reader.Read(grant, policy, aioperation.Outfit)

	if limits.HardCostMicroVND == nil || *limits.HardCostMicroVND != 30000000000 {
		t.Fatalf("the hard cap must come from live configuration, got %v", limits.HardCostMicroVND)
	}
	if limits.EnforcementMode != string(aienforcementmode.ObserveOnly) {
		t.Fatalf("the enforcement mode must come from live configuration, got %q", limits.EnforcementMode)
	}
	if limits.UnknownHoldMinutes != 1440 || limits.MaxUnknownPaidRequestsPerDay != 2 {
		t.Fatalf("the anti-abuse knobs must come from live configuration, got %+v", limits)
	}
}

func TestTokenCeilingsAreReadFromTheSnapshot(t *testing.T) {
	grant := grantWithEntitlements(t, intPtr(4), 6000)
	reader := newReader()

	// Live configuration carries a different number; the bought one must win.
	policy := livePolicy(nil, intPtr(4))
	policy.Operations[0].NormalMaxInputTokens = 999

	limits := reader.Read(grant, policy, aioperation.Outfit)
	if limits.NormalMaxInputTokens != 6000 {
		t.Fatalf("the bought token ceiling must win, got %d", limits.NormalMaxInputTokens)
	}
	if limits.ReducedMaxInputTokens != 5500 {
		t.Fatalf("the bought reduced ceiling must win, got %d", limits.ReducedMaxInputTokens)
	}
}

// TestIncompleteSnapshotIsReportedNotPatched: the eight pre-existing snapshots
// were all incomplete. Treating a missing limit as zero would have frozen the
// free policy's anti-abuse counter at zero — a state that looks valid and
// enforces nothing.
func TestIncompleteSnapshotIsReportedNotPatched(t *testing.T) {
	document, _ := json.Marshal(&PolicySnapshot{
		Envelope:     SnapshotEnvelope{EventVersion: 1},
		Entitlements: map[string]OperationEntitlement{"chat": {MaxPaidAttemptsPerDay: intPtr(3)}},
	})
	grant := &aiexecutor_entities.AIPolicyGrant{
		AuditableEntity: aiexecutor_entities.AuditableEntity{BaseEntity: aiexecutor_entities.BaseEntity{ID: uuid.New()}},
		PolicySnapshot:  aiexecutor_entities.JSONDocument(document),
	}

	limits := newReader().Read(grant, livePolicy(nil, intPtr(4)), aioperation.Outfit)
	if limits.SnapshotComplete {
		t.Fatal("a snapshot missing three operations must be reported incomplete")
	}
	if limits.MaxPaidAttemptsPerDay != nil {
		t.Fatalf("no entitlement may be invented from an incomplete snapshot, got %v", *limits.MaxPaidAttemptsPerDay)
	}
}

func TestEmptySnapshotIsReportedIncomplete(t *testing.T) {
	grant := &aiexecutor_entities.AIPolicyGrant{
		AuditableEntity: aiexecutor_entities.AuditableEntity{BaseEntity: aiexecutor_entities.BaseEntity{ID: uuid.New()}},
		PolicySnapshot:  aiexecutor_entities.JSONDocument(`{}`),
	}
	limits := newReader().Read(grant, livePolicy(nil, intPtr(4)), aioperation.Outfit)
	if limits.SnapshotComplete {
		t.Fatal("an empty snapshot must be reported incomplete")
	}
}

// TestNullAttemptLimitSurvivesTheSnapshot: NULL is a meaningful value in strict
// mode, not missing data. Collapsing it to zero would read as "no paid
// attempts allowed".
func TestNullAttemptLimitSurvivesTheSnapshot(t *testing.T) {
	grant := grantWithEntitlements(t, nil, 6000)
	limits := newReader().Read(grant, livePolicy(int64Ptr(30000000000), intPtr(4)), aioperation.Outfit)
	if !limits.SnapshotComplete {
		t.Fatal("the snapshot is complete")
	}
	if limits.MaxPaidAttemptsPerDay != nil {
		t.Fatalf("a NULL limit must stay NULL, got %d", *limits.MaxPaidAttemptsPerDay)
	}
}

func TestLivePolicyDataIsAlwaysAvailable(t *testing.T) {
	grant := grantWithEntitlements(t, intPtr(4), 6000)
	limits := newReader().Read(grant, livePolicy(int64Ptr(30000000000), intPtr(4)), aioperation.Outfit)

	// Routing is operational, so it comes from the live policy.
	if limits.NormalRoute != string(airoute.Paid) || limits.FreeRoute != string(airoute.Free) {
		t.Fatalf("routes must come from live configuration, got %+v", limits)
	}
	if limits.PeriodDays != 30 {
		t.Fatalf("periodDays must come from live configuration, got %d", limits.PeriodDays)
	}
}
