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
	admin.NewAICostPolicyAdminUseCase,
	orchestrator.NewOrchestratorUseCase,
	grant.NewGrantUseCase,
	// 024 T035: the two-tier degradation resolver and its counter.
	grant.NewGrantFallbackResolver,
	grant.NewFallbackMetrics,
	// 024 T087: the rejection counter, keyed by reason code, so a refused
	// event is visible even though the consumer reports itself healthy and the
	// queue depth is zero.
	grant.NewGrantMetrics,
	// 024 T029b/T065: applies the snapshot/live split at the read path.
	grant.NewEntitlementReader,
	pricing.NewPricingUseCase,
	cost.NewCostUseCase,
)
