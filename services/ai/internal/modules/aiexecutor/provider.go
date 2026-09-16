package aiexecutor

import (
	"smart-wardrobe-be/services/ai/contract/aiexecutor"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/application"
	aiexecutor_infra_caching "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/infrastructure/caching"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/infrastructure/persistence"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/presentation/handler"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/presentation/router"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/presentation/worker"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	application.ProviderSet,

	persistence.NewAICostRepository,
	persistence.NewPricingConfigRepository,
	aiexecutor_infra_caching.NewRedisPricingCacheService,

	contract.NewAIOrchestratorContract,
	contract.NewAICostDashboardReader,

	handler.NewAdminHandler,
	router.NewRouter,

	worker.NewAIUsageReconciliationWorker,
	worker.NewPlanChangedWorker,
)
