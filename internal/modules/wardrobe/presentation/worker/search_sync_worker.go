package worker

import (
	"context"
	"strings"
	"sync/atomic"
	"time"

	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/application/interface/event"
	usecase_interfaces "smart-wardrobe-be/internal/modules/wardrobe/application/interface/usecase"
	"smart-wardrobe-be/internal/pkg/logger"
	"smart-wardrobe-be/internal/shared/observability/workerlog"
	"smart-wardrobe-be/internal/shared/presentation/worker"

	"go.uber.org/zap"
)

type SearchSyncWorker struct {
	*worker.BaseWorker
	eventConsumer   event.ISearchSyncEventConsumer
	useCase         usecase_interfaces.ISearchSyncUseCase
	logger          logger.Interface
	initialSyncDone int32
}

func NewSearchSyncWorker(
	eventConsumer event.ISearchSyncEventConsumer,
	useCase usecase_interfaces.ISearchSyncUseCase,
	l logger.Interface,
) *SearchSyncWorker {
	return &SearchSyncWorker{
		BaseWorker:    worker.NewBaseWorker("search_sync"),
		eventConsumer: eventConsumer,
		useCase:       useCase,
		logger:        l,
	}
}

func (w *SearchSyncWorker) Start(ctx context.Context) {
	w.BaseWorker.Start(ctx)
	go w.manageInitialSyncAndRecovery(ctx)
	go w.startConsume(ctx)
}

func (w *SearchSyncWorker) startConsume(ctx context.Context) {
	err := w.eventConsumer.ConsumeEvents(ctx, func(ctx context.Context, eventPayload dto.WardrobeEventPayload) error {
		run := workerlog.New("search_sync", workerlog.TriggerQueue)
		run.LogReceived(w.logger,
			zap.String("itemId", eventPayload.ItemID.String()),
			zap.String("userId", eventPayload.UserID.String()),
			zap.String("action", eventPayload.Action),
		)
		err := w.useCase.ProcessSyncEvent(ctx, eventPayload, run)
		if err != nil {
			willRequeue := false
			if containsOfflineError(err) {
				willRequeue = true
			}
			run.LogFailure(w.logger, err,
				zap.String("itemId", eventPayload.ItemID.String()),
				zap.String("userId", eventPayload.UserID.String()),
				zap.String("action", eventPayload.Action),
				zap.Bool("willRequeue", willRequeue),
			)
			return err
		}
		run.LogSuccessIfData(w.logger,
			zap.String("itemId", eventPayload.ItemID.String()),
			zap.String("userId", eventPayload.UserID.String()),
			zap.String("action", eventPayload.Action),
		)
		return nil
	})

	if err != nil {
		w.logger.Error("Failed to initiate search sync event consumption process", zap.Error(err))
	}
}

func (w *SearchSyncWorker) manageInitialSyncAndRecovery(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	w.tryInitialSync(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if atomic.LoadInt32(&w.initialSyncDone) == 1 {
				return
			}
			w.tryInitialSync(ctx)
		}
	}
}

func (w *SearchSyncWorker) tryInitialSync(ctx context.Context) {
	run := workerlog.New("search_sync", workerlog.TriggerInitialSync)
	done, err := w.useCase.TryInitialSync(ctx, run)
	if err != nil {
		run.LogFailure(w.logger, err)
	}
	if done {
		run.LogSuccessIfData(w.logger)
		atomic.StoreInt32(&w.initialSyncDone, 1)
	}
}

func containsOfflineError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "elasticsearch is offline")
}
