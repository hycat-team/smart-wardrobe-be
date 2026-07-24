package worker

import (
	"context"
	"time"

	dashboard_app "smart-wardrobe-be/internal/modules/dashboard/application"
	uc_interfaces "smart-wardrobe-be/internal/modules/dashboard/application/interface/usecase"
	"smart-wardrobe-be/internal/pkg/logger"
	"smart-wardrobe-be/internal/shared/observability/workerlog"
)

type DashboardAggregatorWorker struct {
	logger       logger.Interface
	aggregatorUC uc_interfaces.IDashboardAggregatorUseCase
	stopChan     chan struct{}
}

func NewDashboardAggregatorWorker(
	l logger.Interface,
	aggregatorUC uc_interfaces.IDashboardAggregatorUseCase,
) *DashboardAggregatorWorker {
	return &DashboardAggregatorWorker{
		logger:       l,
		aggregatorUC: aggregatorUC,
		stopChan:     make(chan struct{}),
	}
}

func (w *DashboardAggregatorWorker) Start(ctx context.Context) {
	w.logger.Info("Starting Dashboard Aggregator Ticker Worker (interval: 15m)...")
	ticker := time.NewTicker(dashboard_app.AggregationInterval)

	go func() {
		w.runAggregation(ctx)

		for {
			select {
			case <-ctx.Done():
				w.logger.Info("Stopping Dashboard Aggregator Worker via context cancel")
				ticker.Stop()
				return
			case <-w.stopChan:
				w.logger.Info("Stopping Dashboard Aggregator Worker via Stop signal")
				ticker.Stop()
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
	close(w.stopChan)
}
