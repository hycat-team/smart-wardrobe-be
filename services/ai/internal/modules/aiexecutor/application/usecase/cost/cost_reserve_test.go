package cost

import (
	"context"
	"errors"
	"testing"
	"time"

	uc_interfaces "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/application/interface/usecase"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/aioperation"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/airoute"
	aiexecutor_entities "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/entities"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/repositories"

	"github.com/google/uuid"
)

// reserveCaptureRepo records the event handed to Reserve so a free route can be
// asserted to carry no price information at all.
type reserveCaptureRepo struct {
	repositories.IAICostRepository

	event    *aiexecutor_entities.AIUsageEvent
	admitted bool
}

func (r *reserveCaptureRepo) Reserve(
	_ context.Context,
	_ *aiexecutor_entities.AIPolicyGrant,
	_ *aiexecutor_entities.AICostPolicy,
	_ *aiexecutor_entities.AICostPolicyOperation,
	_ *int,
	event *aiexecutor_entities.AIUsageEvent,
	_ time.Time,
) (*aiexecutor_entities.AIUsagePeriodLedger, bool, error) {
	r.event = event
	return &aiexecutor_entities.AIUsagePeriodLedger{}, r.admitted, nil
}

// A free route must not require a pricing row: it reserves nothing, so pricing
// lookup is both unnecessary and, when the FREE model has no row, fatal.
func TestReserveAgainstPolicyFreeRouteSkipsPricing(t *testing.T) {
	repo := &reserveCaptureRepo{admitted: true}
	pricing := &stubPricingUC{err: errors.New("pricing must not be consulted for a free route")}
	uc := &CostUseCase{repo: repo, pricingUC: pricing}

	plan := &uc_interfaces.ExecutionPlan{
		RequestID:             uuid.New(),
		UserID:                uuid.New(),
		Operation:             aioperation.Chat,
		Provider:              "google",
		Model:                 "free-model-without-price",
		Route:                 airoute.Free,
		MaxInputTokens:        3000,
		MaxOutputTokens:       500,
		EstimatedPromptTokens: 120,
	}
	policy := &aiexecutor_entities.AICostPolicy{UnknownHoldMinutes: 15}
	op := &aiexecutor_entities.AICostPolicyOperation{Operation: aioperation.Chat}

	decision, err := uc.reserveAgainstPolicy(context.Background(), plan, time.Now(), nil, policy, op)
	if err != nil {
		t.Fatalf("reserveAgainstPolicy: %v", err)
	}
	if decision.Route != airoute.Free {
		t.Errorf("route = %q, want %q", decision.Route, airoute.Free)
	}
	if pricing.gotProvider != "" || pricing.gotModel != "" {
		t.Errorf("free route must not call pricing, got %q/%q", pricing.gotProvider, pricing.gotModel)
	}
	if repo.event == nil {
		t.Fatal("free route must still create a usage event")
	}
	if repo.event.PricingConfigID != nil {
		t.Errorf("free route pricing_config_id = %v, want nil", repo.event.PricingConfigID)
	}
	if repo.event.ReservedCostMicroVND != 0 || repo.event.EstimatedMaxCostMicroVND != 0 {
		t.Errorf("free route reserved = %d/%d, want 0/0", repo.event.ReservedCostMicroVND, repo.event.EstimatedMaxCostMicroVND)
	}
	if repo.event.InputUSDPerMillion != nil || repo.event.OutputUSDPerMillion != nil || repo.event.USDToVND != nil {
		t.Error("free route must not snapshot price columns")
	}
}
