package worker

import (
	"context"

	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/dto"
	uc_interfaces "smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/interface/usecase"
	"smart-wardrobe-be/shared/application/constants/eventconstants"
	"smart-wardrobe-be/shared/application/event"
	"smart-wardrobe-be/shared/observability/workerlog"
	"smart-wardrobe-be/shared/pkg/logger"
	"smart-wardrobe-be/shared/presentation/worker"

	"go.uber.org/zap"
)

type SearchSyncWorker struct {
	*worker.BaseWorker
	messageClient event.IMessageConsumer
	useCase       uc_interfaces.ISearchSyncUseCase
	logger        logger.Interface
}

func NewSearchSyncWorker(
	messageClient event.IMessageConsumer,
	useCase uc_interfaces.ISearchSyncUseCase,
	l logger.Interface,
) *SearchSyncWorker {
	return &SearchSyncWorker{
		BaseWorker:    worker.NewBaseWorker("search_sync"),
		messageClient: messageClient,
		useCase:       useCase,
		logger:        l,
	}
}

func (w *SearchSyncWorker) Start(ctx context.Context) {
	w.BaseWorker.Start(ctx)
	// FR-001: no backfill when the engine is off. The queue consumer below
	// stays registered so sync events drain with successful no-ops (FR-003)
	// instead of piling up unconsumed.
	if w.useCase.SearchEnabled() {
		go w.useCase.RunInitialSyncWithRetry(w.Context(), w.logger)
	} else {
		w.logger.Info("Search engine disabled — skipping initial sync")
	}
	if err := event.RegisterTypedConsumer(w.messageClient, eventconstants.QueueElasticsearchSync, w.handle, event.ConsumerOpts{}); err != nil {
		w.logger.Error("Failed to register search sync consumer", zap.Error(err))
	}
}

func (w *SearchSyncWorker) handle(ctx context.Context, payload dto.WardrobeEventPayload) error {
	run := workerlog.New("search_sync", workerlog.TriggerQueue)
	run.LogReceived(w.logger,
		zap.String("correlationId", event.CorrelationIdFromContext(ctx)),
		zap.String("itemId", payload.ItemID.String()),
		zap.String("userId", payload.UserID.String()),
		zap.String("action", payload.Action),
	)
	if err := w.useCase.ProcessSyncEvent(ctx, payload, run); err != nil {
		run.LogFailure(w.logger, err,
			zap.String("itemId", payload.ItemID.String()),
			zap.String("userId", payload.UserID.String()),
			zap.String("action", payload.Action),
			zap.Bool("willRequeue", w.useCase.IsOfflineError(err)),
		)
		return err
	}
	run.LogSuccessIfData(w.logger,
		zap.String("itemId", payload.ItemID.String()),
		zap.String("userId", payload.UserID.String()),
		zap.String("action", payload.Action),
	)
	return nil
}

func (w *SearchSyncWorker) Stop() {
	w.BaseWorker.Stop()
}
