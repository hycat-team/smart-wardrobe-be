package aiexecutor

import (
	"smart-wardrobe-be/internal/modules/aiexecutor/application/usecase/aicost"
	"smart-wardrobe-be/internal/modules/aiexecutor/application/usecase/grant"
	"smart-wardrobe-be/internal/modules/aiexecutor/contract"
	"smart-wardrobe-be/internal/modules/aiexecutor/infrastructure/persistence"
	"smart-wardrobe-be/internal/modules/aiexecutor/presentation/worker"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	persistence.NewAICostRepository,
	persistence.NewPricingConfigRepository,

	aicost.NewService,

	contract.NewPricingConfigService,

	grant.NewGrantUseCase,

	worker.NewAIUsageReconciliationWorker,
	worker.NewPlanChangedWorker,
)
