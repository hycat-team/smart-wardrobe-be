package orchestrator

import (
	"context"
	"fmt"
	"time"

	aiexecutor_dto "smart-wardrobe-be/internal/modules/aiexecutor/application/dto"
	"smart-wardrobe-be/internal/modules/aiexecutor/domain/constants/aienforcementmode"
	"smart-wardrobe-be/internal/modules/aiexecutor/domain/constants/aimodel"
	"smart-wardrobe-be/internal/modules/aiexecutor/domain/constants/aiprovider"

	"github.com/google/uuid"
)

// ----- Execution Plan -----

func (uc *OrchestratorUseCase) resolveExecutionPlan(ctx context.Context, userID uuid.UUID, operation string, meta aiexecutor_dto.AITokenEstimationMeta) (*aiexecutor_dto.ExecutionPlan, error) {
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
		Route:                 aiexecutor_dto.AIRouteFree,
		EstimatedPromptTokens: meta.EstimatedPromptTokens,
	}

	if freeOperations[operation] {
		return plan, nil
	}

	grant, policy, op, err := uc.repo.ResolvePolicy(ctx, userID, operation, time.Now())
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

func (uc *OrchestratorUseCase) resolveProviderModel(operation string) (aiprovider.Provider, aimodel.Model) {
	switch operation {
	case aiexecutor_dto.AIOperationChat:
		return aiprovider.Provider(uc.cfg.AI.ChatTextPrimary.Provider), aimodel.Model(uc.cfg.AI.ChatTextPrimary.Model)
	case aiexecutor_dto.AIOperationOutfit:
		return aiprovider.Provider(uc.cfg.AI.RecommendationTextPrimary.Provider), aimodel.Model(uc.cfg.AI.RecommendationTextPrimary.Model)
	case aiexecutor_dto.AIOperationSummary:
		return aiprovider.Provider(uc.cfg.AI.ChatTextPrimary.Provider), aimodel.Model(uc.cfg.AI.ChatTextPrimary.Model)
	case aiexecutor_dto.AIOperationRewriter:
		return aiprovider.Provider(uc.cfg.AI.ChatTextPrimary.Provider), aimodel.Model(uc.cfg.AI.ChatTextPrimary.Model)
	case aiexecutor_dto.AIOperationVision:
		return aiprovider.Provider(uc.cfg.AI.VisionPrimary.Provider), aimodel.Model(uc.cfg.AI.VisionPrimary.Model)
	case aiexecutor_dto.AIOperationEmbedding:
		return aiprovider.Provider(uc.cfg.AI.EmbeddingPrimary.Provider), aimodel.Model(uc.cfg.AI.EmbeddingPrimary.Model)
	default:
		return "", ""
	}
}
