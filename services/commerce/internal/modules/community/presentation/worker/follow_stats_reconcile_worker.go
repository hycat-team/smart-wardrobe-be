package worker

import (
	"context"
	"time"

	uc_interfaces "smart-wardrobe-be/services/commerce/internal/modules/community/application/interface/usecase"
	"smart-wardrobe-be/shared/observability/workerlog"
	"smart-wardrobe-be/shared/pkg/logger"
	"smart-wardrobe-be/shared/presentation/worker"

	"go.uber.org/zap"
)

const reconcileInterval = 30 * time.Minute

// FollowStatsReconcileWorker periodically rebuilds follower/following counters
// from the follows table (denormalized-counter reconciliation).
type FollowStatsReconcileWorker struct {
	*worker.BaseWorker
	useCase uc_interfaces.IStatsSyncUseCase
	logger  logger.Interface
	stopCh  chan struct{}
}

func NewFollowStatsReconcileWorker(useCase uc_interfaces.IStatsSyncUseCase, l logger.Interface) *FollowStatsReconcileWorker {
	return &FollowStatsReconcileWorker{
		BaseWorker: worker.NewBaseWorker("community_follow_stats_reconcile"),
		useCase:    useCase,
		logger:     l,
		stopCh:     make(chan struct{}),
	}
}

func (w *FollowStatsReconcileWorker) Start(ctx context.Context) {
	w.BaseWorker.Start(ctx)
	go w.loop()
}

func (w *FollowStatsReconcileWorker) Stop() {
	select {
	case <-w.stopCh:
	default:
		close(w.stopCh)
	}
	w.BaseWorker.Stop()
}

func (w *FollowStatsReconcileWorker) loop() {
	ticker := time.NewTicker(reconcileInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			w.run()
		case <-w.stopCh:
			return
		}
	}
}

func (w *FollowStatsReconcileWorker) run() {
	ctx, cancel := context.WithTimeout(context.Background(), reconcileInterval)
	defer cancel()

	run := workerlog.New("community_follow_stats_reconcile", workerlog.TriggerCron)
	if err := w.useCase.ReconcileFollowStats(ctx); err != nil {
		run.LogFailure(w.logger, err)
		return
	}
	run.LogSuccessIfData(w.logger, zap.String("status", "reconciled"))
}
