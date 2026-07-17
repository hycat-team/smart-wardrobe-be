package orchestrator

import (
	"smart-wardrobe-be/internal/config"
	aiexecutor_dto "smart-wardrobe-be/internal/modules/aiexecutor/application/dto"
	uc_interfaces "smart-wardrobe-be/internal/modules/aiexecutor/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/aiexecutor/domain/repositories"
	app_ai "smart-wardrobe-be/internal/shared/application/ai"
)

const microVNDPerVND int64 = 1_000_000

var freeOperations = map[string]bool{
	aiexecutor_dto.AIOperationVision:    true,
	aiexecutor_dto.AIOperationEmbedding: true,
}

type OrchestratorUseCase struct {
	aiService app_ai.IAIService
	repo      repositories.IAICostRepository
	cfg       *config.Config
	costUC    uc_interfaces.IAICostUseCase
}

func NewOrchestratorUseCase(
	aiService app_ai.IAIService,
	repo repositories.IAICostRepository,
	cfg *config.Config,
	costUC uc_interfaces.IAICostUseCase,
) uc_interfaces.IAIOrchestratorUseCase {
	return &OrchestratorUseCase{
		aiService: aiService,
		repo:      repo,
		cfg:       cfg,
		costUC:    costUC,
	}
}
