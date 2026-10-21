package orchestrator

import (
	"context"
	"errors"
	"sync"

	aiexecutor_dto "smart-wardrobe-be/services/ai/dto/aiexecutor"
	uc_interfaces "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/application/interface/usecase"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/aioperation"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/airoute"
	app_ai "smart-wardrobe-be/shared/application/ai"
	"smart-wardrobe-be/shared/config"

	"github.com/google/uuid"
)

// ErrLocalFallbackRoute is the sentinel signaled when the orchestrator
// resolves a local route; no remote API call is made.
var ErrLocalFallbackRoute = errors.New("ai route resolved to local fallback: bypassing remote api call")

func mapOptions(opt aiexecutor_dto.OrchestratorOptions, requestID uuid.UUID, onUsage func(app_ai.TokenUsage)) app_ai.TextGenerationOptions {
	return app_ai.TextGenerationOptions{
		MaxOutputTokens:  opt.MaxOutputTokens,
		Temperature:      opt.Temperature,
		ResponseMIMEType: opt.ResponseMIMEType,
		ResponseSchema:   opt.ResponseSchema,
		UserID:           opt.UserID,
		Operation:        opt.Operation,
		RequestID:        requestID,
		OnUsage:          onUsage,
	}
}

// ----- IAIOrchestrator -----

func (uc *OrchestratorUseCase) AnalyzeImage(ctx context.Context, imageUrl string, prompt string) (string, error) {
	plan, err := uc.resolveExecutionPlan(ctx, uuid.Nil, aioperation.Vision, uc_interfaces.AITokenEstimationMeta{})
	if err != nil {
		return "", err
	}
	decision, err := uc.costUC.Prepare(ctx, plan)
	if err != nil {
		return "", err
	}
	text, usage, err := uc.aiService.AnalyzeImageWithUsage(ctx, imageUrl, prompt)
	if err != nil {
		_ = uc.costUC.MarkUnknown(ctx, decision.RequestID, err.Error())
		return "", err
	}
	_ = uc.costUC.Confirm(ctx, decision.RequestID, uc_interfaces.AIUsage{
		PromptTokens:   usage.PromptTokens,
		OutputTokens:   usage.OutputTokens,
		ThinkingTokens: usage.ThinkingTokens,
		FinishReason:   usage.FinishReason,
		Provider:       plan.Provider,
		Model:          plan.Model,
		Unbilled:       uc.cfg.AI.IsUnbilledCapability(config.CapabilityVision),
	})
	return text, nil
}

func (uc *OrchestratorUseCase) GenerateEmbeddings(ctx context.Context, chunks []string) ([][]float32, error) {
	plan, err := uc.resolveExecutionPlan(ctx, uuid.Nil, aioperation.Embedding, uc_interfaces.AITokenEstimationMeta{})
	if err != nil {
		return nil, err
	}
	decision, err := uc.costUC.Prepare(ctx, plan)
	if err != nil {
		return nil, err
	}
	result, err := uc.aiService.GenerateEmbeddings(ctx, chunks)
	if err != nil {
		_ = uc.costUC.MarkUnknown(ctx, decision.RequestID, err.Error())
		return nil, err
	}
	// Embedding has no provider usage signal: batchEmbedContents returns no
	// usage and spike T001 proved countTokens answers 400 for both
	// gemini-embedding-001 and gemini-embedding-2, so no counting call is made
	// here. Confirm with zero tokens and fail open to usage_unreported; the
	// user request still succeeds. If Google supports countTokens for
	// embedding models later, re-enable counting at exactly this point.
	_ = uc.costUC.Confirm(ctx, decision.RequestID, uc_interfaces.AIUsage{
		Provider: plan.Provider,
		Model:    plan.Model,
		Unbilled: uc.cfg.AI.IsUnbilledCapability(config.CapabilityEmbedding),
	})
	return result, nil
}

func (uc *OrchestratorUseCase) GenerateChatText(ctx context.Context, systemPrompt string, userPrompt string, options aiexecutor_dto.OrchestratorOptions) (string, error) {
	est, err := uc.aiService.EstimatePromptTokens(ctx, options.Operation, systemPrompt, userPrompt)
	if err != nil {
		return "", err
	}
	plan, err := uc.resolveExecutionPlan(ctx, options.UserID, aioperation.AIOperation(options.Operation), uc_interfaces.AITokenEstimationMeta{
		EstimatedPromptTokens: est.EstimatedPromptTokens,
		TokenEstimationMethod: est.TokenEstimationMethod,
		TokenCountLatencyMs:   est.TokenCountLatencyMs,
	})
	if err != nil {
		return "", err
	}
	decision, err := uc.costUC.Prepare(ctx, plan)
	if err != nil {
		return "", err
	}

	if decision.Route == airoute.Local {
		_ = uc.costUC.Release(ctx, decision.RequestID, "local fallback triggered")
		return "", ErrLocalFallbackRoute
	}

	if err = uc.costUC.MarkInFlight(ctx, decision.RequestID); err != nil {
		return "", err
	}

	var usage app_ai.TokenUsage
	mappedOpt := mapOptions(options, plan.RequestID, func(u app_ai.TokenUsage) {
		usage = u
	})
	uc.applyTextRoute(plan, &mappedOpt, decision.Route)

	result, err := uc.aiService.GenerateChatText(ctx, systemPrompt, userPrompt, mappedOpt)
	if err != nil {
		_ = uc.costUC.Release(ctx, decision.RequestID, err.Error())
		return "", err
	}

	provider, model := usageProviderModel(plan, usage)
	confirmUsage := uc_interfaces.AIUsage{
		PromptTokens:   usage.PromptTokens,
		OutputTokens:   usage.OutputTokens,
		ThinkingTokens: usage.ThinkingTokens,
		FinishReason:   usage.FinishReason,
		Provider:       provider,
		Model:          model,
		Unbilled:       usage.Unbilled,
	}
	if err = uc.costUC.Confirm(ctx, decision.RequestID, confirmUsage); err != nil {
		return "", err
	}
	return result, nil
}

func (uc *OrchestratorUseCase) GenerateRecommendationText(ctx context.Context, systemPrompt string, userPrompt string, options aiexecutor_dto.OrchestratorOptions) (string, error) {
	est, err := uc.aiService.EstimatePromptTokens(ctx, options.Operation, systemPrompt, userPrompt)
	if err != nil {
		return "", err
	}
	plan, err := uc.resolveExecutionPlan(ctx, options.UserID, aioperation.AIOperation(options.Operation), uc_interfaces.AITokenEstimationMeta{
		EstimatedPromptTokens: est.EstimatedPromptTokens,
		TokenEstimationMethod: est.TokenEstimationMethod,
		TokenCountLatencyMs:   est.TokenCountLatencyMs,
	})
	if err != nil {
		return "", err
	}
	decision, err := uc.costUC.Prepare(ctx, plan)
	if err != nil {
		return "", err
	}

	if decision.Route == airoute.Local {
		_ = uc.costUC.Release(ctx, decision.RequestID, "local fallback triggered")
		return "", ErrLocalFallbackRoute
	}

	if err = uc.costUC.MarkInFlight(ctx, decision.RequestID); err != nil {
		return "", err
	}

	var usage app_ai.TokenUsage
	mappedOpt := mapOptions(options, plan.RequestID, func(u app_ai.TokenUsage) {
		usage = u
	})
	uc.applyTextRoute(plan, &mappedOpt, decision.Route)

	result, err := uc.aiService.GenerateRecommendationText(ctx, systemPrompt, userPrompt, mappedOpt)
	if err != nil {
		_ = uc.costUC.Release(ctx, decision.RequestID, err.Error())
		return "", err
	}

	provider, model := usageProviderModel(plan, usage)
	confirmUsage := uc_interfaces.AIUsage{
		PromptTokens:   usage.PromptTokens,
		OutputTokens:   usage.OutputTokens,
		ThinkingTokens: usage.ThinkingTokens,
		FinishReason:   usage.FinishReason,
		Provider:       provider,
		Model:          model,
		Unbilled:       usage.Unbilled,
	}
	if err = uc.costUC.Confirm(ctx, decision.RequestID, confirmUsage); err != nil {
		return "", err
	}
	return result, nil
}

func (uc *OrchestratorUseCase) GenerateChatTextStream(ctx context.Context, systemPrompt string, userPrompt string, options aiexecutor_dto.OrchestratorOptions) (<-chan string, <-chan error) {
	textChan := make(chan string, 100)
	errChan := make(chan error, 1)

	est, err := uc.aiService.EstimatePromptTokens(ctx, options.Operation, systemPrompt, userPrompt)
	if err != nil {
		errChan <- err
		close(textChan)
		close(errChan)
		return textChan, errChan
	}
	plan, err := uc.resolveExecutionPlan(ctx, options.UserID, aioperation.AIOperation(options.Operation), uc_interfaces.AITokenEstimationMeta{
		EstimatedPromptTokens: est.EstimatedPromptTokens,
		TokenEstimationMethod: est.TokenEstimationMethod,
		TokenCountLatencyMs:   est.TokenCountLatencyMs,
	})
	if err != nil {
		errChan <- err
		close(textChan)
		close(errChan)
		return textChan, errChan
	}
	decision, err := uc.costUC.Prepare(ctx, plan)
	if err != nil {
		errChan <- err
		close(textChan)
		close(errChan)
		return textChan, errChan
	}

	if decision.Route == airoute.Local {
		_ = uc.costUC.Release(ctx, decision.RequestID, "local fallback triggered")
		errChan <- ErrLocalFallbackRoute
		close(textChan)
		close(errChan)
		return textChan, errChan
	}

	if err = uc.costUC.MarkInFlight(ctx, decision.RequestID); err != nil {
		errChan <- err
		close(textChan)
		close(errChan)
		return textChan, errChan
	}

	var mu sync.Mutex
	var usage app_ai.TokenUsage

	mappedOpt := mapOptions(options, plan.RequestID, func(u app_ai.TokenUsage) {
		mu.Lock()
		usage = u
		mu.Unlock()
	})
	uc.applyTextRoute(plan, &mappedOpt, decision.Route)

	aiTextChan, aiErrChan := uc.aiService.GenerateChatTextStream(ctx, systemPrompt, userPrompt, mappedOpt)

	go func() {
		defer close(textChan)
		defer close(errChan)

		for chunk := range aiTextChan {
			textChan <- chunk
		}

		if err, ok := <-aiErrChan; ok {
			_ = uc.costUC.Release(ctx, decision.RequestID, err.Error())
			errChan <- err
			return
		}

		mu.Lock()
		provider, model := usageProviderModel(plan, usage)
		confirmUsage := uc_interfaces.AIUsage{
			PromptTokens:   usage.PromptTokens,
			OutputTokens:   usage.OutputTokens,
			ThinkingTokens: usage.ThinkingTokens,
			FinishReason:   usage.FinishReason,
			Provider:       provider,
			Model:          model,
			Unbilled:       usage.Unbilled,
		}
		mu.Unlock()

		_ = uc.costUC.Confirm(ctx, decision.RequestID, confirmUsage)
	}()

	return textChan, errChan
}

// usageProviderModel prefers the provider that actually served the request over
// the one the plan predicted. They differ whenever a fallback or the free
// failover wins, and the ledger must record what ran, not what was planned.
func usageProviderModel(plan *uc_interfaces.ExecutionPlan, usage app_ai.TokenUsage) (string, string) {
	provider, model := plan.Provider, plan.Model
	if usage.Provider != "" {
		provider = usage.Provider
	}
	if usage.Model != "" {
		model = usage.Model
	}
	return provider, model
}
