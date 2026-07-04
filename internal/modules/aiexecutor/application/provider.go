package application

import (
	"smart-wardrobe-be/internal/modules/aiexecutor/application/usecase/cost"
	"smart-wardrobe-be/internal/modules/aiexecutor/application/usecase/grant"
	"smart-wardrobe-be/internal/modules/aiexecutor/application/usecase/orchestrator"
	"smart-wardrobe-be/internal/modules/aiexecutor/application/usecase/pricing"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	orchestrator.NewOrchestratorUseCase,
	grant.NewGrantUseCase,
	pricing.NewPricingUseCase,
	cost.NewCostUseCase,
)
