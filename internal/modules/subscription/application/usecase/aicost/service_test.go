package aicost

import (
	"context"
	"github.com/google/uuid"
	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/modules/subscription/contract"
	"smart-wardrobe-be/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/entities"
	"smart-wardrobe-be/internal/shared/domain/constants/subscription/aienforcementmode"
	"testing"
	"time"
)

type fakeCostRepo struct {
	grant  *entities.UserAIPolicyGrant
	policy *entities.AICostPolicy
	op     *entities.AICostPolicyOperation
	event  *entities.AIUsageEvent
}

func (f *fakeCostRepo) ResolvePolicy(context.Context, uuid.UUID, string, time.Time) (*entities.UserAIPolicyGrant, *entities.AICostPolicy, *entities.AICostPolicyOperation, error) {
	return f.grant, f.policy, f.op, nil
}
func (f *fakeCostRepo) Reserve(_ context.Context, _ *entities.UserAIPolicyGrant, _ *entities.AICostPolicy, _ *entities.AICostPolicyOperation, e *entities.AIUsageEvent, _ time.Time) (*entities.AIUsagePeriodLedger, bool, error) {
	f.event = e
	return &entities.AIUsagePeriodLedger{}, true, nil
}
func (*fakeCostRepo) MarkInFlight(context.Context, uuid.UUID, time.Time) error { return nil }
func (*fakeCostRepo) Confirm(context.Context, uuid.UUID, int64, int64, int64, int64, string, string, string, time.Time) error {
	return nil
}
func (*fakeCostRepo) Release(context.Context, uuid.UUID, string, time.Time) error     { return nil }
func (*fakeCostRepo) MarkUnknown(context.Context, uuid.UUID, string, time.Time) error { return nil }
func (*fakeCostRepo) ExpireUnknown(context.Context, time.Time, int) (int64, error)    { return 0, nil }

var _ repositories.IAICostRepository = (*fakeCostRepo)(nil)

func TestPremiumOutfitReservationUsesConfiguredDecimalPricing(t *testing.T) {
	hard := int64(25_000_000_000)
	repo := &fakeCostRepo{grant: &entities.UserAIPolicyGrant{AuditableEntity: entities.AuditableEntity{BaseEntity: entities.BaseEntity{ID: uuid.New()}}, UserID: uuid.New()}, policy: &entities.AICostPolicy{HardCostMicroVND: &hard, EnforcementMode: aienforcementmode.Strict, PeriodDays: 30, FreeRouteThresholdBPS: 9200, CompactThresholdBPS: 8000, MaxUnknownPaidRequestsPerDay: 2}, op: &entities.AICostPolicyOperation{Operation: "outfit", NormalRoute: "paid", FreeRoute: "free", NormalMaxInputTokens: 7000, NormalMaxOutputTokens: 400, ReducedMaxInputTokens: 5500, ReducedMaxOutputTokens: 350}}
	cfg := &config.Config{AI: config.AIServiceConfig{Pricing: config.AIPricingConfig{Version: "v1", USDToVND: "27000", Paid: config.AIModelPricingConfig{InputUSDPerMillionTokens: "0.10", OutputUSDPerMillionTokens: "0.40"}}}}
	service := NewService(repo, cfg)
	meta := contract.AITokenEstimationMeta{
		EstimatedPromptTokens: 7000,
		TokenEstimationMethod: contract.TokenEstimationLocal,
		TokenCountLatencyMs:   nil,
	}
	decision, err := service.Prepare(context.Background(), repo.grant.UserID, "outfit", meta)
	if err != nil {
		t.Fatal(err)
	}
	if decision.MaxOutputTokens != 400 {
		t.Fatalf("unexpected output cap %d", decision.MaxOutputTokens)
	}
	if repo.event.ReservedCostMicroVND != 23_220_000 {
		t.Fatalf("expected 23.22 VND reservation, got %d micro-VND", repo.event.ReservedCostMicroVND)
	}
}
