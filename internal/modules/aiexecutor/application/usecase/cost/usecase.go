package cost

import (
	"context"
	"fmt"
	"time"

	aiexecutor_dto "smart-wardrobe-be/internal/modules/aiexecutor/application/dto"
	uc_interfaces "smart-wardrobe-be/internal/modules/aiexecutor/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/aiexecutor/domain/constants/aiusageeventstatus"
	aiexecutor_entities "smart-wardrobe-be/internal/modules/aiexecutor/domain/entities"
	"smart-wardrobe-be/internal/modules/aiexecutor/domain/repositories"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type CostUseCase struct {
	repo      repositories.IAICostRepository
	pricingUC uc_interfaces.IPricingUseCase
}

func NewCostUseCase(
	repo repositories.IAICostRepository,
	pricingUC uc_interfaces.IPricingUseCase,
) uc_interfaces.IAICostUseCase {
	return &CostUseCase{
		repo:      repo,
		pricingUC: pricingUC,
	}
}

var freeOperations = map[string]bool{
	aiexecutor_dto.AIOperationVision:    true,
	aiexecutor_dto.AIOperationEmbedding: true,
}

func (uc *CostUseCase) Prepare(ctx context.Context, plan *aiexecutor_dto.ExecutionPlan) (*aiexecutor_dto.AICostDecision, error) {
	now := time.Now()

	if freeOperations[plan.Operation] {
		event := &aiexecutor_entities.AIUsageEvent{
			RequestID:        plan.RequestID,
			UserID:           plan.UserID,
			Operation:        plan.Operation,
			LogicalRoute:     aiexecutor_dto.AIRouteFree,
			Provider:         &plan.Provider,
			Model:            &plan.Model,
			Status:           aiusageeventstatus.Reserved,
			UnknownExpiresAt: func() *time.Time { t := now.Add(30 * time.Minute); return &t }(),
		}
		if err := uc.repo.CreateEvent(ctx, event); err != nil {
			return nil, err
		}
		return &aiexecutor_dto.AICostDecision{
			RequestID:       plan.RequestID,
			Route:           aiexecutor_dto.AIRouteFree,
			MaxInputTokens:  plan.MaxInputTokens,
			MaxOutputTokens: plan.MaxOutputTokens,
			Tracked:         true,
		}, nil
	}

	grant, policy, op, err := uc.repo.ResolvePolicy(ctx, plan.UserID, plan.Operation, now)
	if err != nil {
		return nil, err
	}
	route := plan.Route
	maxIn, maxOut := plan.MaxInputTokens, plan.MaxOutputTokens

	pricing, err := uc.pricingUC.GetActivePricingConfig(ctx, plan.Provider, plan.Model)
	if err != nil {
		return nil, err
	}

	reserve := int64(0)
	if route == aiexecutor_dto.AIRoutePaid {
		reserve = microCost(plan.EstimatedPromptTokens, int64(maxOut), pricing.InputUSDPerMillion, pricing.OutputUSDPerMillion, pricing.USDToVND)
	}
	unknownExpires := now.Add(time.Duration(policy.UnknownHoldMinutes) * time.Minute)

	event := &aiexecutor_entities.AIUsageEvent{
		RequestID:                plan.RequestID,
		LedgerID:                 uuid.Nil,
		UserID:                   plan.UserID,
		Operation:                plan.Operation,
		LogicalRoute:             route,
		PricingConfigID:          &pricing.ID,
		Provider:                 &plan.Provider,
		Model:                    &plan.Model,
		InputUSDPerMillion:       &pricing.InputUSDPerMillion,
		OutputUSDPerMillion:      &pricing.OutputUSDPerMillion,
		USDToVND:                 &pricing.USDToVND,
		ReservedCostMicroVND:     reserve,
		EstimatedMaxCostMicroVND: reserve,
		Status:                   aiusageeventstatus.Reserved,
		UnknownExpiresAt:         &unknownExpires,
	}

	ledger, admitted, err := uc.repo.Reserve(ctx, grant, policy, op, event, now)
	if err != nil {
		return nil, err
	}
	if !admitted {
		return uc.PrepareFree(ctx, plan.UserID, plan.Operation, 0, maxOut)
	}
	_ = ledger
	return &aiexecutor_dto.AICostDecision{
		RequestID:       plan.RequestID,
		Route:           route,
		MaxInputTokens:  maxIn,
		MaxOutputTokens: maxOut,
		Tracked:         true,
	}, nil
}

func (uc *CostUseCase) PrepareFree(ctx context.Context, userID uuid.UUID, operation string, promptTokens int64, maxOutputTokens int) (*aiexecutor_dto.AICostDecision, error) {
	now := time.Now()
	grant, policy, op, err := uc.repo.ResolvePolicy(ctx, userID, operation, now)
	if err != nil {
		return &aiexecutor_dto.AICostDecision{Route: aiexecutor_dto.AIRouteLocal}, nil
	}
	requestID := uuid.New()
	event := &aiexecutor_entities.AIUsageEvent{
		RequestID:        requestID,
		UserID:           userID,
		Operation:        operation,
		LogicalRoute:     aiexecutor_dto.AIRouteFree,
		PromptTokens:     promptTokens,
		Status:           aiusageeventstatus.Reserved,
		UnknownExpiresAt: func() *time.Time { t := now.Add(time.Duration(policy.UnknownHoldMinutes) * time.Minute); return &t }(),
	}
	_, _, _ = grant, policy, op
	_ = uc.repo.CreateEvent(ctx, event)
	return &aiexecutor_dto.AICostDecision{
		RequestID:       requestID,
		Route:           aiexecutor_dto.AIRouteFree,
		MaxInputTokens:  op.NormalMaxInputTokens,
		MaxOutputTokens: maxOutputTokens,
		Tracked:         true,
	}, nil
}

func (uc *CostUseCase) MarkInFlight(ctx context.Context, requestID uuid.UUID) error {
	return uc.repo.MarkInFlight(ctx, requestID, time.Now())
}

func (uc *CostUseCase) Confirm(ctx context.Context, requestID uuid.UUID, usage aiexecutor_dto.AIUsage) error {
	if usage.Provider == "" && usage.Model == "" {
		_ = uc.repo.Confirm(ctx, requestID, 0, 0, 0, 0, "", "", "", time.Now())
		return nil
	}

	pricing, err := uc.pricingUC.GetActivePricingConfig(ctx, usage.Provider, usage.Model)
	if err != nil {
		return fmt.Errorf("cannot find pricing for actual provider %s / model %s: %w", usage.Provider, usage.Model, err)
	}

	cost := microCost(usage.PromptTokens, usage.OutputTokens+usage.ThinkingTokens, pricing.InputUSDPerMillion, pricing.OutputUSDPerMillion, pricing.USDToVND)
	return uc.repo.Confirm(ctx, requestID, usage.PromptTokens, usage.OutputTokens, usage.ThinkingTokens, cost, string(usage.Provider), string(usage.Model), usage.FinishReason, time.Now())
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

func microCost(inputTokens, outputTokens int64, inputRate, outputRate, fx decimal.Decimal) int64 {
	return decimal.NewFromInt(inputTokens).Mul(inputRate).Mul(fx).Add(decimal.NewFromInt(outputTokens).Mul(outputRate).Mul(fx)).Ceil().IntPart()
}
