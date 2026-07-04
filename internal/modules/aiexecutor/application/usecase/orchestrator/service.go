package orchestrator

import (
	"context"
	"fmt"
	"time"

	"smart-wardrobe-be/config"
	aiexecutor_dto "smart-wardrobe-be/internal/modules/aiexecutor/application/dto"
	"smart-wardrobe-be/internal/modules/aiexecutor/contract"
	"smart-wardrobe-be/internal/modules/aiexecutor/domain/constants/aienforcementmode"
	"smart-wardrobe-be/internal/modules/aiexecutor/domain/constants/aiusageeventstatus"
	aiexecutor_entities "smart-wardrobe-be/internal/modules/aiexecutor/domain/entities"
	"smart-wardrobe-be/internal/modules/aiexecutor/domain/repositories"
	app_ai "smart-wardrobe-be/internal/shared/application/ai"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

const microVNDPerVND int64 = 1_000_000

type UseCase struct {
	aiService app_ai.IAIService
	repo      repositories.IAICostRepository
	cfg       *config.Config
}

func NewOrchestratorUseCase(aiService app_ai.IAIService, repo repositories.IAICostRepository, cfg *config.Config) *UseCase {
	return &UseCase{aiService: aiService, repo: repo, cfg: cfg}
}

func mapOptions(opt contract.OrchestratorOptions) app_ai.TextGenerationOptions {
	return app_ai.TextGenerationOptions{
		MaxOutputTokens:  opt.MaxOutputTokens,
		Temperature:      opt.Temperature,
		ResponseMIMEType: opt.ResponseMIMEType,
		ResponseSchema:   opt.ResponseSchema,
		UserID:           opt.UserID,
		Operation:        opt.Operation,
	}
}

func (uc *UseCase) AnalyzeImage(ctx context.Context, imageUrl string, prompt string) (string, error) {
	return uc.aiService.AnalyzeImage(ctx, imageUrl, prompt)
}

func (uc *UseCase) GenerateEmbeddings(ctx context.Context, chunks []string) ([][]float32, error) {
	return uc.aiService.GenerateEmbeddings(ctx, chunks)
}

func (uc *UseCase) GenerateChatText(ctx context.Context, systemPrompt string, userPrompt string, options contract.OrchestratorOptions) (string, error) {
	return uc.aiService.GenerateChatText(ctx, systemPrompt, userPrompt, mapOptions(options))
}

func (uc *UseCase) GenerateRecommendationText(ctx context.Context, systemPrompt string, userPrompt string, options contract.OrchestratorOptions) (string, error) {
	return uc.aiService.GenerateRecommendationText(ctx, systemPrompt, userPrompt, mapOptions(options))
}

func (uc *UseCase) GenerateChatTextStream(ctx context.Context, systemPrompt string, userPrompt string, options contract.OrchestratorOptions) (<-chan string, <-chan error) {
	return uc.aiService.GenerateChatTextStream(ctx, systemPrompt, userPrompt, mapOptions(options))
}

// ----- IAICostUseCase -----

func (uc *UseCase) rates() (decimal.Decimal, decimal.Decimal, decimal.Decimal) {
	in, _ := decimal.NewFromString(uc.cfg.AI.FallbackPricing.Paid.InputUSDPerMillionTokens)
	out, _ := decimal.NewFromString(uc.cfg.AI.FallbackPricing.Paid.OutputUSDPerMillionTokens)
	fx, _ := decimal.NewFromString(uc.cfg.AI.FallbackPricing.USDToVND)
	return in, out, fx
}

func microCost(inputTokens, outputTokens int64, inputRate, outputRate, fx decimal.Decimal) int64 {
	return decimal.NewFromInt(inputTokens).Mul(inputRate).Mul(fx).Add(decimal.NewFromInt(outputTokens).Mul(outputRate).Mul(fx)).Ceil().IntPart()
}

func (uc *UseCase) Prepare(ctx context.Context, userID uuid.UUID, operation string, meta aiexecutor_dto.AITokenEstimationMeta) (*aiexecutor_dto.AICostDecision, error) {
	if meta.EstimatedPromptTokens < 0 {
		return nil, fmt.Errorf("estimated prompt tokens cannot be negative: %d", meta.EstimatedPromptTokens)
	}
	if meta.TokenEstimationMethod != aiexecutor_dto.TokenEstimationLocal && meta.TokenEstimationMethod != aiexecutor_dto.TokenEstimationProviderCount {
		return nil, fmt.Errorf("invalid token estimation method: %s", meta.TokenEstimationMethod)
	}
	if meta.TokenEstimationMethod == aiexecutor_dto.TokenEstimationProviderCount && meta.TokenCountLatencyMs == nil {
		return nil, fmt.Errorf("token count latency must be provided when estimation method is PROVIDER_COUNT")
	}
	if meta.TokenEstimationMethod == aiexecutor_dto.TokenEstimationLocal && meta.TokenCountLatencyMs != nil {
		return nil, fmt.Errorf("token count latency must be nil when estimation method is LOCAL")
	}

	now := time.Now()
	grant, policy, op, err := uc.repo.ResolvePolicy(ctx, userID, operation, now)
	if err != nil {
		return nil, err
	}
	route := op.NormalRoute
	maxIn, maxOut := op.NormalMaxInputTokens, op.NormalMaxOutputTokens
	if policy.EnforcementMode == aienforcementmode.FreeOnly {
		route = op.FreeRoute
	}
	inputRate, outputRate, fx := uc.rates()
	reserve := int64(0)
	if route == aiexecutor_dto.AIRoutePaid {
		reserve = microCost(meta.EstimatedPromptTokens, int64(maxOut), inputRate, outputRate, fx)
	}
	requestID := uuid.New()
	unknownExpires := now.Add(time.Duration(policy.UnknownHoldMinutes) * time.Minute)

	event := &aiexecutor_entities.AIUsageEvent{
		RequestID:                requestID,
		UserID:                   userID,
		Operation:                operation,
		LogicalRoute:             route,
		InputUSDPerMillion:       &inputRate,
		OutputUSDPerMillion:      &outputRate,
		USDToVND:                 &fx,
		PromptTokens:             0,
		EstimatedPromptTokens:    &meta.EstimatedPromptTokens,
		TokenEstimationMethod:    &meta.TokenEstimationMethod,
		TokenCountLatencyMs:      meta.TokenCountLatencyMs,
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
		return uc.PrepareFree(ctx, userID, operation, meta.EstimatedPromptTokens, op.NormalMaxOutputTokens)
	}
	_ = ledger
	return &aiexecutor_dto.AICostDecision{
		RequestID:       requestID,
		Route:           route,
		MaxInputTokens:  maxIn,
		MaxOutputTokens: maxOut,
		Tracked:         true,
	}, nil
}

func (uc *UseCase) PrepareFree(ctx context.Context, userID uuid.UUID, operation string, promptTokens int64, maxOutputTokens int) (*aiexecutor_dto.AICostDecision, error) {
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

func (uc *UseCase) MarkInFlight(ctx context.Context, requestID uuid.UUID) error {
	return uc.repo.MarkInFlight(ctx, requestID, time.Now())
}

func (uc *UseCase) Confirm(ctx context.Context, requestID uuid.UUID, usage aiexecutor_dto.AIUsage) error {
	in, out, fx := uc.rates()
	cost := microCost(usage.PromptTokens, usage.OutputTokens+usage.ThinkingTokens, in, out, fx)
	return uc.repo.Confirm(ctx, requestID, usage.PromptTokens, usage.OutputTokens, usage.ThinkingTokens, cost, usage.Provider, usage.Model, usage.FinishReason, time.Now())
}

func (uc *UseCase) Release(ctx context.Context, requestID uuid.UUID, reason string) error {
	return uc.repo.Release(ctx, requestID, reason, time.Now())
}

func (uc *UseCase) MarkUnknown(ctx context.Context, requestID uuid.UUID, reason string) error {
	return uc.repo.MarkUnknown(ctx, requestID, reason, time.Now().Add(time.Hour))
}

func (uc *UseCase) ExpireUnknown(ctx context.Context, now time.Time, limit int) (int64, error) {
	return uc.repo.ExpireUnknown(ctx, now, limit)
}
