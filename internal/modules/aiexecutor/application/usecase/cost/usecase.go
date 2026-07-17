package cost

import (
	"context"
	"fmt"
	"time"

	aiexecutor_dto "smart-wardrobe-be/internal/modules/aiexecutor/application/dto"
	uc_interfaces "smart-wardrobe-be/internal/modules/aiexecutor/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/aiexecutor/domain/constants/aioperation"
	"smart-wardrobe-be/internal/modules/aiexecutor/domain/constants/airoute"
	"smart-wardrobe-be/internal/modules/aiexecutor/domain/constants/aiusageeventstatus"
	aiexecutor_entities "smart-wardrobe-be/internal/modules/aiexecutor/domain/entities"
	"smart-wardrobe-be/internal/modules/aiexecutor/domain/repositories"
	subscription_contract "smart-wardrobe-be/internal/modules/subscription/presentation/contract"

	"github.com/google/uuid"
)

type CostUseCase struct {
	repo            repositories.IAICostRepository
	pricingUC       uc_interfaces.IPricingUseCase
	userSubContract subscription_contract.IUserSubscriptionContract
}

func NewCostUseCase(
	repo repositories.IAICostRepository,
	pricingUC uc_interfaces.IPricingUseCase,
	userSubContract subscription_contract.IUserSubscriptionContract,
) uc_interfaces.IAICostUseCase {
	return &CostUseCase{
		repo:            repo,
		pricingUC:       pricingUC,
		userSubContract: userSubContract,
	}
}

var freeOperations = map[aioperation.AIOperation]bool{
	aioperation.Vision:    true,
	aioperation.Embedding: true,
}

func (uc *CostUseCase) Prepare(ctx context.Context, plan *aiexecutor_dto.ExecutionPlan) (*aiexecutor_dto.AICostDecision, error) {
	now := time.Now()

	if freeOperations[plan.Operation] {
		event := &aiexecutor_entities.AIUsageEvent{
			RequestID: plan.RequestID,
			UserID: func() *uuid.UUID {
				if plan.UserID == uuid.Nil {
					return nil
				}
				return &plan.UserID
			}(),
			Operation:        plan.Operation,
			LogicalRoute:     airoute.Free,
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
			Route:           airoute.Free,
			MaxInputTokens:  plan.MaxInputTokens,
			MaxOutputTokens: plan.MaxOutputTokens,
			Tracked:         true,
		}, nil
	}

	subOverview, err := uc.userSubContract.GetUserSubscriptionOverview(ctx, plan.UserID)
	if err != nil {
		return nil, err
	}

	grant, policy, op, err := uc.repo.ResolvePolicy(ctx, subOverview.PlanID, plan.UserID, plan.Operation, now)
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
	if route == airoute.Paid {
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

func (uc *CostUseCase) PrepareFree(ctx context.Context, userID uuid.UUID, operation aioperation.AIOperation, promptTokens int64, maxOutputTokens int) (*aiexecutor_dto.AICostDecision, error) {
	now := time.Now()
	subOverview, err := uc.userSubContract.GetUserSubscriptionOverview(ctx, userID)
	if err != nil {
		return &aiexecutor_dto.AICostDecision{Route: airoute.Local}, nil
	}
	grant, policy, op, err := uc.repo.ResolvePolicy(ctx, subOverview.PlanID, userID, operation, now)
	if err != nil {
		return &aiexecutor_dto.AICostDecision{Route: airoute.Local}, nil
	}
	requestID := uuid.New()
	event := &aiexecutor_entities.AIUsageEvent{
		RequestID: requestID,
		UserID: func() *uuid.UUID {
			if userID == uuid.Nil {
				return nil
			}
			return &userID
		}(),
		Operation:        operation,
		LogicalRoute:     airoute.Free,
		PromptTokens:     promptTokens,
		Status:           aiusageeventstatus.Reserved,
		UnknownExpiresAt: func() *time.Time { t := now.Add(time.Duration(policy.UnknownHoldMinutes) * time.Minute); return &t }(),
	}
	_, _, _ = grant, policy, op
	_ = uc.repo.CreateEvent(ctx, event)
	return &aiexecutor_dto.AICostDecision{
		RequestID:       requestID,
		Route:           airoute.Free,
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
	return uc.repo.Confirm(ctx, requestID, usage.PromptTokens, usage.OutputTokens, usage.ThinkingTokens, cost, usage.Provider, usage.Model, usage.FinishReason, time.Now())
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
