package worker

import (
	"context"
	"time"

	dashboard_app "smart-wardrobe-be/services/commerce/internal/modules/dashboard/application"
	uc_interfaces "smart-wardrobe-be/services/commerce/internal/modules/dashboard/application/interface/usecase"
	"smart-wardrobe-be/shared/observability/workerlog"
	"smart-wardrobe-be/shared/pkg/logger"
	"smart-wardrobe-be/shared/presentation/worker"

	"go.uber.org/zap"
)

type DashboardAggregatorWorker struct {
	*worker.BaseWorker
	logger       logger.Interface
	aggregatorUC uc_interfaces.IDashboardAggregatorUseCase
}

func NewDashboardAggregatorWorker(
	l logger.Interface,
	aggregatorUC uc_interfaces.IDashboardAggregatorUseCase,
) *DashboardAggregatorWorker {
	return &DashboardAggregatorWorker{
		BaseWorker:   worker.NewBaseWorker("dashboard_aggregator"),
		logger:       l,
		aggregatorUC: aggregatorUC,
	}
}

func (w *DashboardAggregatorWorker) Start(ctx context.Context) {
	w.BaseWorker.Start(ctx)
	w.logger.Info("Starting Dashboard Aggregator Ticker Worker", zap.Duration("interval", dashboard_app.AggregationInterval))

	go func() {
		ticker := time.NewTicker(dashboard_app.AggregationInterval)
		defer ticker.Stop()

		w.runAggregation(ctx)

		for {
			select {
			case <-ctx.Done():
				w.logger.Info("Stopping Dashboard Aggregator Worker")
				return
			case <-ticker.C:
				w.runAggregation(ctx)
			}
		}
	}()
}

func (w *DashboardAggregatorWorker) runAggregation(ctx context.Context) {
	run := workerlog.New("dashboard_aggregator", workerlog.TriggerCron)
	if err := w.aggregatorUC.AggregateAdminSnapshots(ctx, run); err != nil {
		run.LogFailure(w.logger, err)
		return
	}
	if err := w.aggregatorUC.AggregateBrandSnapshots(ctx, run); err != nil {
		run.LogFailure(w.logger, err)
		return
	}
	run.LogSuccessIfData(w.logger)
}

func (w *DashboardAggregatorWorker) Stop() {
	w.BaseWorker.Stop()
}
