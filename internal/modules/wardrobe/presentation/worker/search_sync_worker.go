package worker

import (
	"context"
	"sync/atomic"
	"time"

	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/application/interface/event"
	usecase_interfaces "smart-wardrobe-be/internal/modules/wardrobe/application/interface/usecase"
	"smart-wardrobe-be/pkg/logger"

	"go.uber.org/zap"
)

type SearchSyncWorker struct {
	eventConsumer   event.ISearchSyncEventConsumer
	useCase         usecase_interfaces.ISearchSyncUseCase
	logger          logger.Interface
	initialSyncDone int32 // 0 = not done, 1 = done
}

func NewSearchSyncWorker(
	eventConsumer event.ISearchSyncEventConsumer,
	useCase usecase_interfaces.ISearchSyncUseCase,
	l logger.Interface,
) *SearchSyncWorker {
	w := &SearchSyncWorker{
		eventConsumer:   eventConsumer,
		useCase:         useCase,
		logger:          l,
		initialSyncDone: 0,
	}

	// Manage initial sync and recovery of search index in background loop
	go w.manageInitialSyncAndRecovery()

	// Start listening to the sync event queue via the Application layer Consumer
	go w.startConsume()

	return w
}

func (w *SearchSyncWorker) startConsume() {
	ctx := context.Background()
	err := w.eventConsumer.ConsumeEvents(ctx, func(ctx context.Context, eventPayload dto.WardrobeEventPayload) error {
		return w.useCase.ProcessSyncEvent(ctx, eventPayload)
	})

	if err != nil {
		w.logger.Error("Failed to initiate search sync event consumption process", zap.Error(err))
	}
}

func (w *SearchSyncWorker) manageInitialSyncAndRecovery() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	w.tryInitialSync()

	for range ticker.C {
		if atomic.LoadInt32(&w.initialSyncDone) == 1 {
			return
		}
		w.tryInitialSync()
	}
}

func (w *SearchSyncWorker) tryInitialSync() {
	ctx := context.Background()
	done, err := w.useCase.TryInitialSync(ctx)
	if err != nil {
		w.logger.Warn("[SearchSyncWorker] Error during initial sync attempt", zap.Error(err))
	}
	if done {
		atomic.StoreInt32(&w.initialSyncDone, 1)
	}
}
