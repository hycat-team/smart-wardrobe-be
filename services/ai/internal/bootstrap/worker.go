package bootstrap

import (
	"context"

	aiworker "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/presentation/worker"
	sharedWorker "smart-wardrobe-be/shared/presentation/worker"
)

// Worker is the lifecycle contract every background worker satisfies through
// the shared BaseWorker (FR-044/FR-045).
type Worker interface {
	Start(ctx context.Context)
	Stop()
}

// Workers groups the AI Engine background workers. They run only in
// APP_MODE=worker|all; APP_MODE=http never starts them (FR-044, SC-018).
type Workers struct {
	AIUsageReconciliation *aiworker.AIUsageReconciliationWorker
	PlanChanged           *aiworker.PlanChangedWorker
	Outbox                *sharedWorker.OutboxWorker
}

// All returns the workers in a stable start/stop order.
func (w *Workers) All() []Worker {
	if w == nil {
		return nil
	}
	return []Worker{
		w.AIUsageReconciliation,
		w.PlanChanged,
		w.Outbox,
	}
}
