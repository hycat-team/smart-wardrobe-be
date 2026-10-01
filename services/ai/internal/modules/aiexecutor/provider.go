package aiexecutor

import (
	"smart-wardrobe-be/services/ai/contract/aiexecutor"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/application"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/application/usecase/grant"
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
	persistence.NewAICostPolicyAuditRepository,
	persistence.NewPricingConfigRepository,
	aiexecutor_infra_caching.NewRedisPricingCacheService,

	contract.NewAIOrchestratorContract,
	contract.NewAICostDashboardReader,

	handler.NewAdminHandler,
	handler.NewCostPolicyAdminHandler,
	router.NewRouter,
	// 024 T034/T035: the degradation counter satisfies the resolver's observer
	// port, so no use case has to know which concrete type records degradations.
	wire.Bind(new(grant.FallbackObserver), new(*grant.FallbackMetrics)),

	worker.NewAIUsageReconciliationWorker,
	worker.NewPlanChangedWorker,
)
