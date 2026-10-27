package worker

import (
	"context"
	"errors"

	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	uc_interfaces "smart-wardrobe-be/services/commerce/internal/modules/fashion/application/interface/usecase"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/usecase/personalization/learning"
	wardrobe_dto "smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/shared/application/constants/eventconstants"
	"smart-wardrobe-be/shared/application/event"
	"smart-wardrobe-be/shared/observability/workerlog"
	"smart-wardrobe-be/shared/pkg/logger"
	"smart-wardrobe-be/shared/presentation/worker"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// StyleLearningWorker consumes wardrobe.outfit.saved from style_learning_queue
// and applies it to the user's learned style profile (feature 033, US4).
//
// Learning is a cross-module write side effect of Wardrobe, so it arrives as an
// outbox event instead of a synchronous call (§III). The queue is at-least-once:
// the use case drops a duplicate signalId, which is what makes the consumer
// idempotent (FR-037).
type StyleLearningWorker struct {
	*worker.BaseWorker
	messageClient event.IMessageConsumer
	useCase       uc_interfaces.IStyleLearningUseCase
	logger        logger.Interface
}

func NewStyleLearningWorker(
	messageClient event.IMessageConsumer,
	useCase uc_interfaces.IStyleLearningUseCase,
	l logger.Interface,
) *StyleLearningWorker {
	return &StyleLearningWorker{
		BaseWorker:    worker.NewBaseWorker("style_learning"),
		messageClient: messageClient,
		useCase:       useCase,
		logger:        l,
	}
}

func (w *StyleLearningWorker) Start(ctx context.Context) {
	w.BaseWorker.Start(ctx)
	// Prefetch/concurrency come from the Topology registry
	// (style_learning_queue: delivery limit 5, DLQ style_learning_queue.dlq).
	if err := event.RegisterTypedConsumer(w.messageClient, eventconstants.QueueStyleLearning, w.handle, event.ConsumerOpts{}); err != nil {
		w.logger.Error("Failed to register style learning consumer", zap.Error(err))
	}
}

// handle maps the Wardrobe event payload onto the learning input and classifies
// the outcome for the broker:
//
//   - a malformed signal (missing ids, unknown source) is permanent -> DLQ,
//     because no number of redeliveries can learn it;
//   - anything else (database, repository) is retryable -> requeue within the
//     queue's delivery limit, then DLQ.
func (w *StyleLearningWorker) handle(ctx context.Context, payload wardrobe_dto.OutfitSavedEventPayload) error {
	run := workerlog.New("style_learning_apply_signal", workerlog.TriggerQueue)
	run.LogReceived(w.logger,
		zap.String("correlationId", event.CorrelationIdFromContext(ctx)),
		zap.String("signalId", payload.SignalID.String()),
		zap.String("userId", payload.UserID.String()),
		zap.String("outfitId", payload.OutfitID.String()),
		zap.String("source", payload.Source),
		zap.Int("fashionItemCount", len(payload.FashionItemIDs)),
	)

	if err := w.useCase.ApplySignal(ctx, toStyleSignalInput(payload)); err != nil {
		run.LogFailure(w.logger, err,
			zap.String("signalId", payload.SignalID.String()),
			zap.String("userId", payload.UserID.String()),
			zap.Error(err),
		)
		if errors.Is(err, learning.ErrInvalidSignal) {
			return event.PermanentError{Err: err}
		}
		return event.RetryableError{Err: err}
	}

	run.LogSuccessIfData(w.logger,
		zap.String("signalId", payload.SignalID.String()),
		zap.String("userId", payload.UserID.String()),
	)
	return nil
}

// toStyleSignalInput adapts the Wardrobe event DTO to the shared learning input.
// Only the Wardrobe event DTO crosses the module boundary here - no entity and
// no repository is imported from Wardrobe (§II).
func toStyleSignalInput(payload wardrobe_dto.OutfitSavedEventPayload) dto.StyleSignalInput {
	var outfitID *uuid.UUID
	if payload.OutfitID != uuid.Nil {
		value := payload.OutfitID
		outfitID = &value
	}

	return dto.StyleSignalInput{
		SignalID:       payload.SignalID,
		UserID:         payload.UserID,
		Source:         payload.Source,
		OutfitID:       outfitID,
		FashionItemIDs: payload.FashionItemIDs,
		OccurredAt:     payload.OccurredAt,
	}
}

func (w *StyleLearningWorker) Stop() {
	w.BaseWorker.Stop()
}
