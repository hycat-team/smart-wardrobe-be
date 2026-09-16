package orchestrator

import (
	uc_interfaces "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/application/interface/usecase"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/aioperation"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/repositories"
	app_ai "smart-wardrobe-be/shared/application/ai"
	"smart-wardrobe-be/shared/config"
)

const microVNDPerVND int64 = 1_000_000

var freeOperations = map[aioperation.AIOperation]bool{
	aioperation.Vision:    true,
	aioperation.Embedding: true,
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
