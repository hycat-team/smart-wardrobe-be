package worker

import (
	"context"
	"time"

	dashboard_app "smart-wardrobe-be/internal/modules/dashboard/application"
	uc_interfaces "smart-wardrobe-be/internal/modules/dashboard/application/interface/usecase"
	"smart-wardrobe-be/internal/pkg/logger"

	"go.uber.org/zap"
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
				if err := w.aggregatorUC.AggregateAdminSnapshots(ctx); err != nil {
					w.logger.Error("Failed to aggregate admin snapshots", zap.Error(err))
				}
				if err := w.aggregatorUC.AggregateBrandSnapshots(ctx); err != nil {
					w.logger.Error("Failed to aggregate brand snapshots", zap.Error(err))
				}
			}
		}
	}()
}

func (w *DashboardAggregatorWorker) Stop() {
	close(w.stopChan)
}
