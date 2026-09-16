package application

import (
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/application/usecase/admin"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/application/usecase/cost"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/application/usecase/grant"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/application/usecase/orchestrator"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/application/usecase/pricing"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	admin.NewAIAdminUseCase,
	orchestrator.NewOrchestratorUseCase,
	grant.NewGrantUseCase,
	pricing.NewPricingUseCase,
	cost.NewCostUseCase,
)
