package aicost

import (
	"context"
	"fmt"
	"time"

	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/modules/subscription/contract"
	"smart-wardrobe-be/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/entities"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

const microVNDPerVND int64 = 1_000_000

type Service struct {
	repo repositories.IAICostRepository
	cfg  *config.Config
}

func NewService(repo repositories.IAICostRepository, cfg *config.Config) contract.IAICostPolicyContract {
	return &Service{repo: repo, cfg: cfg}
}

func (s *Service) rates() (decimal.Decimal, decimal.Decimal, decimal.Decimal) {
	in, _ := decimal.NewFromString(s.cfg.AI.Pricing.Paid.InputUSDPerMillionTokens)
	out, _ := decimal.NewFromString(s.cfg.AI.Pricing.Paid.OutputUSDPerMillionTokens)
	fx, _ := decimal.NewFromString(s.cfg.AI.Pricing.USDToVND)
	return in, out, fx
}

// microCost deliberately keeps the calculation in decimal. The million-token divisor
// and the million-micro-VND multiplier cancel each other out.
func microCost(inputTokens, outputTokens int64, inputRate, outputRate, fx decimal.Decimal) int64 {
	return decimal.NewFromInt(inputTokens).Mul(inputRate).Mul(fx).Add(decimal.NewFromInt(outputTokens).Mul(outputRate).Mul(fx)).Ceil().IntPart()
}

func (s *Service) Prepare(ctx context.Context, userID uuid.UUID, operation string, meta contract.AITokenEstimationMeta) (*contract.AICostDecision, error) {
	// Validate Caller invariants
	if meta.EstimatedPromptTokens < 0 {
		return nil, fmt.Errorf("estimated prompt tokens cannot be negative: %d", meta.EstimatedPromptTokens)
	}
	if meta.TokenEstimationMethod != contract.TokenEstimationLocal && meta.TokenEstimationMethod != contract.TokenEstimationProviderCount {
		return nil, fmt.Errorf("invalid token estimation method: %s", meta.TokenEstimationMethod)
	}
	if meta.TokenEstimationMethod == contract.TokenEstimationProviderCount && meta.TokenCountLatencyMs == nil {
		return nil, fmt.Errorf("token count latency must be provided when estimation method is PROVIDER_COUNT")
	}
	if meta.TokenEstimationMethod == contract.TokenEstimationLocal && meta.TokenCountLatencyMs != nil {
		return nil, fmt.Errorf("token count latency must be nil when estimation method is LOCAL")
	}

	now := time.Now()
	grant, policy, op, err := s.repo.ResolvePolicy(ctx, userID, operation, now)
	if err != nil {
		return nil, err
	}
	route := op.NormalRoute
	maxIn, maxOut := op.NormalMaxInputTokens, op.NormalMaxOutputTokens
	if policy.EnforcementMode == "FREE_ONLY" {
		route = op.FreeRoute
	}
	inputRate, outputRate, fx := s.rates()
	reserve := int64(0)
	if route == contract.AIRoutePaid {
		reserve = microCost(meta.EstimatedPromptTokens, int64(maxOut), inputRate, outputRate, fx)
	}
	requestID := uuid.New()
	pricingVersion := s.cfg.AI.Pricing.Version
	unknownExpires := now.Add(time.Duration(policy.UnknownHoldMinutes) * time.Minute)
	
	event := &entities.AIUsageEvent{
		RequestID:                requestID, 
		UserID:                   userID, 
		Operation:                operation, 
		LogicalRoute:             route, 
		PricingVersion:           &pricingVersion, 
		InputUSDPerMillion:       &inputRate, 
		OutputUSDPerMillion:      &outputRate, 
		USDToVND:                 &fx, 
		PromptTokens:             0, // updates to actual tokens on Confirm()
		EstimatedPromptTokens:    &meta.EstimatedPromptTokens,
		TokenEstimationMethod:    &meta.TokenEstimationMethod,
		TokenCountLatencyMs:      meta.TokenCountLatencyMs,
		ReservedCostMicroVND:     reserve, 
		EstimatedMaxCostMicroVND: reserve, 
		Status:                   "RESERVED", 
		UnknownExpiresAt:         &unknownExpires,
	}
	
	ledger, admitted, err := s.repo.Reserve(ctx, grant, policy, op, event, now)
	if err != nil {
		return nil, err
	}
	if !admitted {
		return s.PrepareFree(ctx, userID, operation, meta.EstimatedPromptTokens, op.NormalMaxOutputTokens)
	}
	if admitted && policy.EnforcementMode == "STRICT" && policy.HardCostMicroVND != nil {
		ratio := int64(policy.FreeRouteThresholdBPS)
		if ratio <= 0 {
			ratio = 10000
		}
		threshold := *policy.HardCostMicroVND * ratio / 10000
		if ledger.ActualCostMicroVND+ledger.ReservedCostMicroVND >= threshold {
			maxIn, maxOut = op.ReducedMaxInputTokens, op.ReducedMaxOutputTokens
		}
	}
	return &contract.AICostDecision{RequestID: requestID, Route: route, MaxInputTokens: maxIn, MaxOutputTokens: maxOut, Tracked: admitted}, nil
}

func (s *Service) PrepareFree(ctx context.Context, userID uuid.UUID, operation string, promptTokens int64, maxOutputTokens int) (*contract.AICostDecision, error) {
	now := time.Now()
	grant, policy, op, err := s.repo.ResolvePolicy(ctx, userID, operation, now)
	if err != nil {
		return nil, err
	}
	requestID := uuid.New()
	unknownExpires := now.Add(time.Duration(policy.UnknownHoldMinutes) * time.Minute)
	event := &entities.AIUsageEvent{RequestID: requestID, UserID: userID, Operation: operation, LogicalRoute: contract.AIRouteFree, PromptTokens: promptTokens, Status: "RESERVED", UnknownExpiresAt: &unknownExpires}
	_, admitted, err := s.repo.Reserve(ctx, grant, policy, op, event, now)
	if err != nil {
		return nil, err
	}
	if !admitted {
		return &contract.AICostDecision{Route: contract.AIRouteLocal}, nil
	}
	return &contract.AICostDecision{RequestID: requestID, Route: contract.AIRouteFree, MaxInputTokens: op.NormalMaxInputTokens, MaxOutputTokens: maxOutputTokens, Tracked: true}, nil
}
func (s *Service) MarkInFlight(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return nil
	}
	return s.repo.MarkInFlight(ctx, id, time.Now())
}
func (s *Service) Confirm(ctx context.Context, id uuid.UUID, u contract.AIUsage) error {
	if id == uuid.Nil {
		return nil
	}
	in, out, fx := s.rates()
	cost := microCost(u.PromptTokens, u.OutputTokens+u.ThinkingTokens, in, out, fx)
	return s.repo.Confirm(ctx, id, u.PromptTokens, u.OutputTokens, u.ThinkingTokens, cost, u.Provider, u.Model, u.FinishReason, time.Now())
}
func (s *Service) Release(ctx context.Context, id uuid.UUID, reason string) error {
	if id == uuid.Nil {
		return nil
	}
	return s.repo.Release(ctx, id, reason, time.Now())
}
func (s *Service) MarkUnknown(ctx context.Context, id uuid.UUID, reason string) error {
	if id == uuid.Nil {
		return nil
	}
	return s.repo.MarkUnknown(ctx, id, reason, time.Time{})
}
func (s *Service) ExpireUnknown(ctx context.Context, now time.Time, limit int) (int64, error) {
	return s.repo.ExpireUnknown(ctx, now, limit)
}

var _ = microVNDPerVND
