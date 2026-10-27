package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/usecase/personalization/learning"
	wardrobe_dto "smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/shared/application/constants/eventconstants"
	"smart-wardrobe-be/shared/application/event"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type nopWorkerLogger struct{}

func (nopWorkerLogger) Debug(string, ...zap.Field) {}
func (nopWorkerLogger) Info(string, ...zap.Field)  {}
func (nopWorkerLogger) Warn(string, ...zap.Field)  {}
func (nopWorkerLogger) Error(string, ...zap.Field) {}
func (nopWorkerLogger) Fatal(string, ...zap.Field) {}

// fakeConsumer captures the typed handler registered through
// event.RegisterTypedConsumer so a test can deliver a message body to it.
type fakeConsumer struct {
	queue   string
	handler func(context.Context, event.Message) error
	opts    event.ConsumerOpts
}

func (c *fakeConsumer) Consume(string) (<-chan event.Message, error) { return nil, nil }

func (c *fakeConsumer) RegisterConsumer(string, func(event.Message) error) {}

func (c *fakeConsumer) RegisterTypedHandler(
	queueName string,
	handler func(context.Context, event.Message) error,
	opts event.ConsumerOpts,
) error {
	c.queue = queueName
	c.handler = handler
	c.opts = opts
	return nil
}

func (c *fakeConsumer) Close() {}

type fakeLearningUseCase struct {
	inputs []dto.StyleSignalInput
	err    error
}

func (f *fakeLearningUseCase) ApplySignal(_ context.Context, input dto.StyleSignalInput) error {
	f.inputs = append(f.inputs, input)
	return f.err
}

func newStyleLearningWorkerFixture() (*StyleLearningWorker, *fakeConsumer, *fakeLearningUseCase) {
	consumer := &fakeConsumer{}
	useCase := &fakeLearningUseCase{}
	worker := NewStyleLearningWorker(consumer, useCase, nopWorkerLogger{})
	worker.Start(context.Background())
	return worker, consumer, useCase
}

func deliver(t *testing.T, consumer *fakeConsumer, payload any) error {
	t.Helper()
	if consumer.handler == nil {
		t.Fatal("the worker did not register a consumer")
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	return consumer.handler(context.Background(), event.Message{Body: body})
}

func TestStyleLearningWorkerConsumesOutfitSaved(t *testing.T) {
	worker, consumer, useCase := newStyleLearningWorkerFixture()

	if consumer.queue != eventconstants.QueueStyleLearning {
		t.Fatalf("queue = %q, want %q", consumer.queue, eventconstants.QueueStyleLearning)
	}

	signalID := uuid.New()
	userID := uuid.New()
	outfitID := uuid.New()
	itemIDs := []uuid.UUID{uuid.New(), uuid.New()}
	occurredAt := time.Date(2026, 10, 27, 9, 0, 0, 0, time.UTC)

	payload := wardrobe_dto.OutfitSavedEventPayload{
		SignalID:       signalID,
		UserID:         userID,
		OutfitID:       outfitID,
		FashionItemIDs: itemIDs,
		Source:         "outfit_saved",
		OccurredAt:     occurredAt,
	}
	if err := deliver(t, consumer, payload); err != nil {
		t.Fatalf("handle: %v", err)
	}

	if len(useCase.inputs) != 1 {
		t.Fatalf("ApplySignal calls = %d, want 1", len(useCase.inputs))
	}
	input := useCase.inputs[0]
	if input.SignalID != signalID || input.UserID != userID {
		t.Fatalf("input ids = %s/%s, want %s/%s", input.SignalID, input.UserID, signalID, userID)
	}
	if input.OutfitID == nil || *input.OutfitID != outfitID {
		t.Fatalf("outfit id = %v, want %s", input.OutfitID, outfitID)
	}
	if len(input.FashionItemIDs) != 2 || input.FashionItemIDs[0] != itemIDs[0] {
		t.Fatalf("fashion item ids = %v, want %v", input.FashionItemIDs, itemIDs)
	}
	if input.Source != "outfit_saved" {
		t.Fatalf("source = %q, want outfit_saved", input.Source)
	}
	if !input.OccurredAt.Equal(occurredAt) {
		t.Fatalf("occurred at = %v, want %v", input.OccurredAt, occurredAt)
	}

	worker.Stop()
}

// A missing outfit id is legal (the signal is history); the adapter must leave
// the pointer nil instead of storing the zero UUID.
func TestStyleLearningWorkerOmitsMissingOutfitID(t *testing.T) {
	_, consumer, useCase := newStyleLearningWorkerFixture()

	if err := deliver(t, consumer, wardrobe_dto.OutfitSavedEventPayload{
		SignalID:       uuid.New(),
		UserID:         uuid.New(),
		FashionItemIDs: []uuid.UUID{uuid.New()},
		Source:         "outfit_saved",
	}); err != nil {
		t.Fatalf("handle: %v", err)
	}

	if len(useCase.inputs) != 1 || useCase.inputs[0].OutfitID != nil {
		t.Fatalf("outfit id = %v, want nil", useCase.inputs[0].OutfitID)
	}
}

// A repository/DB failure must be requeued, not dead-lettered.
func TestStyleLearningWorkerRetriesTransientErrors(t *testing.T) {
	_, consumer, useCase := newStyleLearningWorkerFixture()
	useCase.err = fmt.Errorf("learn: %w", errors.New("database is unavailable"))

	err := deliver(t, consumer, wardrobe_dto.OutfitSavedEventPayload{
		SignalID:       uuid.New(),
		UserID:         uuid.New(),
		FashionItemIDs: []uuid.UUID{uuid.New()},
		Source:         "outfit_saved",
	})

	var retryable event.RetryableError
	if !errors.As(err, &retryable) {
		t.Fatalf("error = %v, want event.RetryableError", err)
	}
	var permanent event.PermanentError
	if errors.As(err, &permanent) {
		t.Fatal("a transient error must not be classified as permanent")
	}
}

// A malformed signal can never be learned: it goes to the DLQ.
func TestStyleLearningWorkerDeadLettersInvalidSignal(t *testing.T) {
	_, consumer, useCase := newStyleLearningWorkerFixture()
	useCase.err = learning.ErrInvalidSignal

	err := deliver(t, consumer, wardrobe_dto.OutfitSavedEventPayload{})

	var permanent event.PermanentError
	if !errors.As(err, &permanent) {
		t.Fatalf("error = %v, want event.PermanentError", err)
	}
}

// Broken JSON is turned into a permanent error by RegisterTypedConsumer.
func TestStyleLearningWorkerDeadLettersMalformedJSON(t *testing.T) {
	_, consumer, _ := newStyleLearningWorkerFixture()

	if consumer.handler == nil {
		t.Fatal("the worker did not register a consumer")
	}
	err := consumer.handler(context.Background(), event.Message{Body: []byte("{not json")})

	var permanent event.PermanentError
	if !errors.As(err, &permanent) {
		t.Fatalf("error = %v, want event.PermanentError", err)
	}
}
