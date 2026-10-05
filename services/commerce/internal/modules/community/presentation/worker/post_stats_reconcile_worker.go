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

const postReconcileInterval = 30 * time.Minute

// PostStatsReconcileWorker periodically rebuilds the denormalized like/comment
// counters on posts from the likes and comments tables (feature 022 R12).
type PostStatsReconcileWorker struct {
	*worker.BaseWorker
	useCase uc_interfaces.IStatsSyncUseCase
	logger  logger.Interface
	stopCh  chan struct{}
}

func NewPostStatsReconcileWorker(useCase uc_interfaces.IStatsSyncUseCase, l logger.Interface) *PostStatsReconcileWorker {
	return &PostStatsReconcileWorker{
		BaseWorker: worker.NewBaseWorker("community_post_stats_reconcile"),
		useCase:    useCase,
		logger:     l,
		stopCh:     make(chan struct{}),
	}
}

func (w *PostStatsReconcileWorker) Start(ctx context.Context) {
	w.BaseWorker.Start(ctx)
	go w.loop()
}

func (w *PostStatsReconcileWorker) Stop() {
	select {
	case <-w.stopCh:
	default:
		close(w.stopCh)
	}
	w.BaseWorker.Stop()
}

func (w *PostStatsReconcileWorker) loop() {
	ticker := time.NewTicker(postReconcileInterval)
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

func (w *PostStatsReconcileWorker) run() {
	ctx, cancel := context.WithTimeout(context.Background(), postReconcileInterval)
	defer cancel()

	run := workerlog.New("community_post_stats_reconcile", workerlog.TriggerCron)
	if err := w.useCase.ReconcilePostCounters(ctx); err != nil {
		run.LogFailure(w.logger, err)
		return
	}
	run.LogSuccessIfData(w.logger, zap.String("status", "reconciled"))
}
