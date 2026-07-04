package aiexecutor

import (
	"smart-wardrobe-be/internal/modules/aiexecutor/application"
	"smart-wardrobe-be/internal/modules/aiexecutor/contract"
	"smart-wardrobe-be/internal/modules/aiexecutor/infrastructure/persistence"
	"smart-wardrobe-be/internal/modules/aiexecutor/presentation/worker"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	application.ProviderSet,

	persistence.NewAICostRepository,
	persistence.NewPricingConfigRepository,

	contract.NewPricingConfigService,

	worker.NewAIUsageReconciliationWorker,
	worker.NewPlanChangedWorker,
)
