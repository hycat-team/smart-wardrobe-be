package aiexecutor

import (
	"smart-wardrobe-be/internal/modules/aiexecutor/application"
	aiexecutor_infra_caching "smart-wardrobe-be/internal/modules/aiexecutor/infrastructure/caching"
	"smart-wardrobe-be/internal/modules/aiexecutor/infrastructure/persistence"
	"smart-wardrobe-be/internal/modules/aiexecutor/presentation/contract"
	"smart-wardrobe-be/internal/modules/aiexecutor/presentation/worker"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	application.ProviderSet,

	persistence.NewAICostRepository,
	persistence.NewPricingConfigRepository,
	aiexecutor_infra_caching.NewRedisPricingCacheService,

	contract.NewAIOrchestratorContract,
	contract.NewAICostDashboardReader,

	worker.NewAIUsageReconciliationWorker,
	worker.NewPlanChangedWorker,
)
