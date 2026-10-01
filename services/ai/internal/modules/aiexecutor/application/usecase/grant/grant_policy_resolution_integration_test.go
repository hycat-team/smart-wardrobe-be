package grant

import (
	"context"
	"testing"
	"time"

	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/aienforcementmode"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/aioperation"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/airoute"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/entities"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/infrastructure/persistence"
	"smart-wardrobe-be/shared/domain/constants/aipolicygrantstatus"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// 024 T165 / FR-029b: the live is-active flag of a policy, and the live enabled flag of
// an operation, must be read FOR ENFORCEMENT. Enforcing them is what makes disabling
// take effect on an account that already holds a grant — and those accounts are exactly
// the ones a grant cannot fix on its own, because a lifetime grant is only rewritten
// when the account changes plan.
//
// Before this, ResolvePolicy filtered on the operation flag but not the policy flag, so
// `ResolvedLimits.IsActive` was carried, never read, and deactivating a purchased plan's
// cost policy changed nothing.
//
// These tests live in this package, NOT next to the repository, on purpose. They need a
// real database, and the only tables-truncating convention in the AI service is here
// (`cleanupGrantTables`). A second copy of that helper in another package would truncate
// the same tables from a test binary Go runs in parallel, and the two would delete each
// other's fixtures — which is exactly what happened the first time these tests were
// written in the persistence package.

// seedPolicyWithFlags writes a policy and one operation row with the flags under test.
func seedPolicyWithFlags(t *testing.T, conn *gorm.DB, policyActive, operationEnabled bool) *entities.AICostPolicy {
	t.Helper()

	policy := &entities.AICostPolicy{
		Code:            "premium-" + uuid.NewString()[:8],
		Version:         1,
		Name:            "premium",
		EnforcementMode: aienforcementmode.Strict,
		PeriodDays:      30,
		IsActive:        policyActive,
	}
	policy.ID = uuid.New()
	policy.Operations = []*entities.AICostPolicyOperation{
		{
			PolicyID:             policy.ID,
			Operation:            aioperation.Outfit,
			IsEnabled:            operationEnabled,
			NormalRoute:          airoute.Paid,
			FreeRoute:            airoute.Free,
			NormalMaxInputTokens: 6000,
		},
	}
	if err := conn.Create(policy).Error; err != nil {
		t.Fatalf("seed policy: %v", err)
	}
	return policy
}

// seedGrantForPolicy writes a live grant in force right now for the given policy.
func seedGrantForPolicy(t *testing.T, conn *gorm.DB, policyID uuid.UUID) uuid.UUID {
	t.Helper()

	userID := uuid.New()
	grant := &entities.AIPolicyGrant{
		UserID:         userID,
		PolicyID:       policyID,
		PlanID:         uuid.New(),
		PlanCode:       "premium",
		PolicySnapshot: entities.JSONDocument(`{}`),
		EffectiveFrom:  time.Now().UTC().Add(-time.Hour),
		Status:         aipolicygrantstatus.Active,
	}
	grant.ID = uuid.New()
	if err := conn.Create(grant).Error; err != nil {
		t.Fatalf("create grant: %v", err)
	}
	return userID
}

func TestDeactivatedPolicyStopsGranting(t *testing.T) {
	dbConn := openGrantTestDB(t)
	cleanupGrantTables(t, dbConn)

	repo := persistence.NewAICostRepository(dbConn)

	// Control: with the policy active the grant resolves, so the test's premise holds.
	activePolicy := seedPolicyWithFlags(t, dbConn, true, true)
	activeUser := seedGrantForPolicy(t, dbConn, activePolicy.ID)

	grant, policy, op, err := repo.ResolvePolicy(context.Background(), activeUser, aioperation.Outfit, time.Now())
	if err != nil {
		t.Fatalf("an active policy must resolve: %v", err)
	}
	if grant == nil || policy == nil || op == nil {
		t.Fatal("test premise broken: an active policy must resolve")
	}

	// Now deactivate that same policy. This is the operator's lever, and it must take
	// effect on an account that already holds a live grant.
	if err := dbConn.Model(&entities.AICostPolicy{}).
		Where("id = ?", activePolicy.ID).
		Update("is_active", false).Error; err != nil {
		t.Fatalf("deactivate policy: %v", err)
	}

	grant, resolvedPolicy, op, err := repo.ResolvePolicy(context.Background(), activeUser, aioperation.Outfit, time.Now())
	if err != nil {
		t.Fatalf("deactivating must not surface as an error, got: %v", err)
	}
	if grant != nil || resolvedPolicy != nil || op != nil {
		t.Fatalf("deactivating the policy must stop entitlement, got grant %v / policy %v / op %v",
			grant != nil, resolvedPolicy != nil, op != nil)
	}
}

// The operation flag was already enforced; this pins it next to the policy flag so the
// two cannot drift apart — the whole finding was that one was enforced and the other
// looked identical in the struct.
func TestDisabledOperationStopsGranting(t *testing.T) {
	dbConn := openGrantTestDB(t)
	cleanupGrantTables(t, dbConn)

	repo := persistence.NewAICostRepository(dbConn)

	policy := seedPolicyWithFlags(t, dbConn, true, false)
	userID := seedGrantForPolicy(t, dbConn, policy.ID)

	grant, resolvedPolicy, op, err := repo.ResolvePolicy(context.Background(), userID, aioperation.Outfit, time.Now())
	if err != nil {
		t.Fatalf("a disabled operation must not surface as an error, got: %v", err)
	}
	if grant != nil || resolvedPolicy != nil || op != nil {
		t.Fatalf("a disabled operation must stop entitlement, got grant %v / policy %v / op %v",
			grant != nil, resolvedPolicy != nil, op != nil)
	}
}
