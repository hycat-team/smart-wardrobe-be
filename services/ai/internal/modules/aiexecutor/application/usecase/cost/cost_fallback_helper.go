package cost

import (
	"context"
	"time"

	uc_interfaces "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/application/interface/usecase"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/application/usecase/grant"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/aioperation"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/airoute"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/aiusageeventstatus"
	aiexecutor_entities "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/entities"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/repositories"

	"github.com/google/uuid"
)

// tier2TokenCeiling bounds the token budget of a tier-2 request.
//
// The numbers live in the grant package next to FallbackResolution so the
// orchestrator and this use case cannot disagree about the degraded budget.
const (
	tier2MaxInputTokens  = grant.Tier2MaxInputTokens
	tier2MaxOutputTokens = grant.Tier2MaxOutputTokens
)

// defaultUnknownHoldMinutes is used when no policy row is readable, so a
// reserved cost always has an expiry. Without it an interrupted request would
// hold its reservation forever and permanently consume part of the cycle budget.
const defaultUnknownHoldMinutes = 1440

// buildTier2Decision produces the cost decision for a request that has no usable
// policy at all (024 FR-007 tier 2).
//
// It deliberately writes NO ledger row and calls no Reserve. That is not an
// omission — it is the requirement. The read path must not change the
// policy-grant data (FR-042), and a "temporary" grant record created during a
// lookup would bring that record back, which is exactly the mechanism GĐ6 exists
// to delete. Instead the request reuses the free path that already exists: an
// AIUsageEvent with a NULL ledger_id, a free route, and Tracked set, which is
// what Vision and Embedding use today.
//
// The trade-off is accepted deliberately: there is no reservation, so a tier-2
// request cannot overshoot. That is safe precisely because tier 2 permits zero
// paid attempts, so there is no paid budget to overshoot. Verification data
// still lands in ai_usage_events with ledger_id IS NULL, and the degradation is
// counted separately (FR-010).
func buildTier2Decision(
	ctx context.Context,
	repo repositories.IAICostRepository,
	requestID uuid.UUID,
	userID uuid.UUID,
	operation aioperation.AIOperation,
	provider string,
	model string,
	promptTokens int64,
	now time.Time,
) (*uc_interfaces.AICostDecision, error) {
	userIDPtr := func() *uuid.UUID {
		if userID == uuid.Nil {
			return nil
		}
		return &userID
	}()

	event := &aiexecutor_entities.AIUsageEvent{
		RequestID:    requestID,
		LedgerID:     nil,
		UserID:       userIDPtr,
		Operation:    operation,
		LogicalRoute: airoute.Free,
		Provider:     &provider,
		Model:        &model,
		PromptTokens: promptTokens,
		Status:       aiusageeventstatus.Reserved,
		// No unknown-hold window: there is no reserved cost to reconcile, so
		// there is nothing that could end up unknown.
	}
	if err := repo.CreateEvent(ctx, event); err != nil {
		return nil, err
	}

	return &uc_interfaces.AICostDecision{
		RequestID:       requestID,
		Route:           airoute.Free,
		MaxInputTokens:  tier2MaxInputTokens,
		MaxOutputTokens: tier2MaxOutputTokens,
		Tracked:         true,
	}, nil
}
