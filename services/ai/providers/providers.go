// Package providers exposes the AI Engine service's composition surface for
// the transitional monolith (internal/di, internal/bootstrap, internal/api).
//
// It re-exports the service's wire ProviderSet plus the concrete router and
// worker types the monolith composes. It MUST NOT import implementation
// packages of other services. Removed/replaced by per-service composition
// (T042) when cmd/server leaves main at T034.
package providers

import (
	aicontract "smart-wardrobe-be/services/ai/contract/aiexecutor"
	aiexecutormodule "smart-wardrobe-be/services/ai/internal/modules/aiexecutor"
	aiexecutorrouter "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/presentation/router"
	aiexecutorworker "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/presentation/worker"

	"github.com/google/wire"
)

// ProviderSet aggregates the AI Engine module (aiexecutor).
var ProviderSet = wire.NewSet(
	aiexecutormodule.ProviderSet,
)

// Router aliases for monolith composition.
type (
	AIExecutorRouter = aiexecutorrouter.Router
)

// Worker aliases for monolith composition.
type (
	AIUsageReconciliationWorker = aiexecutorworker.AIUsageReconciliationWorker
	PlanChangedWorker           = aiexecutorworker.PlanChangedWorker
)

// AIBundle is the composed product of the AI Engine service: its router and
// workers plus the cross-service contracts it owns.
type AIBundle struct {
	AIExecutorRouter *AIExecutorRouter

	AIUsageReconciliationWorker *AIUsageReconciliationWorker
	PlanChangedWorker           *PlanChangedWorker

	AIOrchestrator        aicontract.IAIOrchestrator
	AICostDashboardReader aicontract.IAICostDashboardReader
}
