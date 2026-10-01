package orchestrator

import (
	uc_interfaces "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/application/interface/usecase"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/application/usecase/grant"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/repositories"
	app_ai "smart-wardrobe-be/shared/application/ai"
	"smart-wardrobe-be/shared/config"
	"smart-wardrobe-be/shared/pkg/logger"
)

const microVNDPerVND int64 = 1_000_000

type OrchestratorUseCase struct {
	aiService        app_ai.IAIService
	repo             repositories.IAICostRepository
	cfg              *config.Config
	costUC           uc_interfaces.IAICostUseCase
	fallbackResolver *grant.GrantFallbackResolver
	// 024 T065: applies the snapshot/live split when building the plan.
	entitlements *grant.EntitlementReader
	logger       logger.Interface
}

func NewOrchestratorUseCase(
	aiService app_ai.IAIService,
	repo repositories.IAICostRepository,
	cfg *config.Config,
	costUC uc_interfaces.IAICostUseCase,
	fallbackResolver *grant.GrantFallbackResolver,
	entitlements *grant.EntitlementReader,
	log logger.Interface,
) uc_interfaces.IAIOrchestratorUseCase {
	return &OrchestratorUseCase{
		aiService:        aiService,
		repo:             repo,
		cfg:              cfg,
		costUC:           costUC,
		fallbackResolver: fallbackResolver,
		entitlements:     entitlements,
		logger:           log,
	}
}
