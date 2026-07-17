package orchestrator

import (
	"context"
	"fmt"
	"time"

	aiexecutor_dto "smart-wardrobe-be/internal/modules/aiexecutor/application/dto"
	"smart-wardrobe-be/internal/modules/aiexecutor/domain/constants/aienforcementmode"
	"smart-wardrobe-be/internal/modules/aiexecutor/domain/constants/aioperation"
	"smart-wardrobe-be/internal/modules/aiexecutor/domain/constants/airoute"

	"github.com/google/uuid"
)

// ----- Execution Plan -----

func (uc *OrchestratorUseCase) resolveExecutionPlan(ctx context.Context, userID uuid.UUID, operation aioperation.AIOperation, meta aiexecutor_dto.AITokenEstimationMeta) (*aiexecutor_dto.ExecutionPlan, error) {
	provider, model := uc.resolveProviderModel(operation)
	if model == "" {
		return nil, fmt.Errorf("unsupported operation: %s", operation)
	}

	plan := &aiexecutor_dto.ExecutionPlan{
		RequestID:             uuid.New(),
		UserID:                userID,
		Operation:             operation,
		Provider:              provider,
		Model:                 model,
		Route:                 airoute.Free,
		EstimatedPromptTokens: meta.EstimatedPromptTokens,
	}

	if freeOperations[operation] {
		return plan, nil
	}

	grant, policy, op, err := uc.repo.ResolvePolicy(ctx, uuid.Nil, userID, operation, time.Now())
	if err != nil {
		return nil, err
	}
	_ = grant

	plan.Route = op.NormalRoute
	plan.MaxInputTokens = op.NormalMaxInputTokens
	plan.MaxOutputTokens = op.NormalMaxOutputTokens

	if policy.EnforcementMode == aienforcementmode.FreeOnly {
		plan.Route = op.FreeRoute
		plan.MaxInputTokens = op.ReducedMaxInputTokens
		plan.MaxOutputTokens = op.ReducedMaxOutputTokens
	}

	return plan, nil
}

func (uc *OrchestratorUseCase) resolveProviderModel(operation aioperation.AIOperation) (string, string) {
	switch operation {
	case aioperation.Chat:
		return uc.cfg.AI.ChatTextPrimary.Provider, uc.cfg.AI.ChatTextPrimary.Model
	case aioperation.Outfit:
		return uc.cfg.AI.RecommendationTextPrimary.Provider, uc.cfg.AI.RecommendationTextPrimary.Model
	case aioperation.Summary:
		return uc.cfg.AI.ChatTextPrimary.Provider, uc.cfg.AI.ChatTextPrimary.Model
	case aioperation.Rewriter:
		return uc.cfg.AI.ChatTextPrimary.Provider, uc.cfg.AI.ChatTextPrimary.Model
	case aioperation.Vision:
		return uc.cfg.AI.VisionPrimary.Provider, uc.cfg.AI.VisionPrimary.Model
	case aioperation.Embedding:
		return uc.cfg.AI.EmbeddingPrimary.Provider, uc.cfg.AI.EmbeddingPrimary.Model
	default:
		return "", ""
	}
}
