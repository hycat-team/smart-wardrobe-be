package application

import (
	uc_interfaces "smart-wardrobe-be/internal/modules/aiexecutor/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/aiexecutor/application/usecase/grant"
	"smart-wardrobe-be/internal/modules/aiexecutor/application/usecase/orchestrator"
	"smart-wardrobe-be/internal/modules/aiexecutor/contract"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	orchestrator.NewOrchestratorUseCase,
	grant.NewGrantUseCase,
	wire.Bind(new(contract.IAIOrchestrator), new(*orchestrator.UseCase)),
	wire.Bind(new(uc_interfaces.IAICostUseCase), new(*orchestrator.UseCase)),
)
