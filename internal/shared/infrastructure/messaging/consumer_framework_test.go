package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"smart-wardrobe-be/internal/shared/application/event"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

// nopLogger là logger.Interface tối giản cho test, thu thập error message
// để assert logging side-effect.
type nopLogger struct{ errs []string }

func (l *nopLogger) Debug(string, ...zap.Field) {}
func (l *nopLogger) Info(string, ...zap.Field)  {}
func (l *nopLogger) Warn(string, ...zap.Field)  {}
func (l *nopLogger) Fatal(string, ...zap.Field) {}
func (l *nopLogger) Error(m string, _ ...zap.Field) {
	l.errs = append(l.errs, m)
}

func newTestClient(t *testing.T) *RabbitMQClient {
	t.Helper()
	baseCtx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	return &RabbitMQClient{
		logger:   &nopLogger{},
		baseCtx:  baseCtx,
		baseCancel: cancel,
	}
}

func TestTopologyConcurrencyDefaults(t *testing.T) {
	if got := topologyConcurrency(QueueFashionAnalyze); got != 3 {
		t.Errorf("fashion_analyze concurrency = %d, want 3 (from Topology)", got)
	}
	if got := topologyConcurrency(QueueElasticsearchSync); got != 1 {
		t.Errorf("elasticsearch_sync concurrency = %d, want 1", got)
	}
	if got := topologyConcurrency("unknown_queue"); got != 1 {
		t.Errorf("unknown queue concurrency = %d, want 1 (fallback)", got)
	}
}

func TestTopologyPrefetchDefaults(t *testing.T) {
	if got := topologyPrefetch(QueueFashionAnalyze); got != 20 {
		t.Errorf("prefetch = %d, want 20 (from Topology)", got)
	}
	if got := topologyPrefetch("unknown_queue"); got != 20 {
		t.Errorf("unknown queue prefetch = %d, want 20 (fallback)", got)
	}
}

func TestRegisterTypedConsumer_UnmarshalFailIsPermanent(t *testing.T) {
	client := newTestClient(t)
	handlerCalled := false

	err := RegisterTypedConsumer(client, QueueWardrobeCleanup, func(ctx context.Context, payload event.Message) error {
		handlerCalled = true
		return nil
	}, ConsumerOpts{})
	if err != nil {
		t.Fatalf("RegisterTypedConsumer returned error: %v", err)
	}
	if len(client.consumers) != 1 {
		t.Fatalf("consumers registered = %d, want 1", len(client.consumers))
	}

	wrapper := client.consumers[0].handler
	werr := wrapper(event.Message{Body: []byte("{not-json")})
	if werr == nil {
		t.Fatal("wrapper returned nil error for poison JSON")
	}
	var pe PermanentError
	if !errors.As(werr, &pe) {
		t.Fatalf("error is not PermanentError: %v", werr)
	}
	if handlerCalled {
		t.Error("handler must not be called on unmarshal failure")
	}
}

func TestRegisterTypedConsumer_CorrelationIdInContext(t *testing.T) {
	client := newTestClient(t)
	type payload struct {
		ItemID uuid.UUID `json:"item_id"`
	}
	correlationID := "corr-123"

	var gotCtx context.Context
	err := RegisterTypedConsumer(client, QueueWardrobeCleanup, func(ctx context.Context, p payload) error {
		gotCtx = ctx
		return nil
	}, ConsumerOpts{})

	if err != nil {
		t.Fatalf("RegisterTypedConsumer returned error: %v", err)
	}
	body, _ := json.Marshal(payload{ItemID: uuid.New()})
	msg := event.Message{
		Body:    body,
		Headers: map[string]any{"x-correlation-id": correlationID},
	}
	if err := client.consumers[0].handler(msg); err != nil {
		t.Fatalf("wrapper returned error: %v", err)
	}
	if got := CorrelationIdFromContext(gotCtx); got != correlationID {
		t.Errorf("correlationId in ctx = %q, want %q", got, correlationID)
	}
	if gotCtx == nil {
		t.Error("handler must receive a non-nil ctx derived from client baseCtx")
	}
}

func TestRegisterTypedConsumer_ConcurrencyFromTopology(t *testing.T) {
	client := newTestClient(t)
	err := RegisterTypedConsumer(client, QueueFashionAnalyze, func(ctx context.Context, payload struct{}) error {
		return nil
	}, ConsumerOpts{})
	if err != nil {
		t.Fatalf("RegisterTypedConsumer returned error: %v", err)
	}
	if len(client.consumers) != 3 {
		t.Errorf("consumers registered = %d, want 3 (Topology concurrency for fashion_analyze)", len(client.consumers))
	}
}

func TestErrorTaxonomyClassification(t *testing.T) {
	// Mô phỏng switch trong startConsumerLoop:
	// Permanent → Nack(false,false) → DLQ; Retryable → Nack(false,true) → requeue.
	permanent := PermanentError{Err: errors.New("bad json")}
	retryable := RetryableError{Err: errors.New("elasticsearch is offline")}

	var pe PermanentError
	var re RetryableError
	if !errors.As(permanent, &pe) {
		t.Error("PermanentError should match errors.As PermanentError")
	}
	if errors.As(permanent, &re) {
		t.Error("PermanentError must not match RetryableError")
	}
	if !errors.As(retryable, &re) {
		t.Error("RetryableError should match errors.As RetryableError")
	}
	if errors.As(retryable, &pe) {
		t.Error("RetryableError must not match PermanentError")
	}
}

func TestAMQPTableToMap(t *testing.T) {
	if got := amqpTableToMap(nil); got != nil {
		t.Errorf("amqpTableToMap(nil) = %v, want nil", got)
	}
	table := amqp.Table{"x-correlation-id": "abc", "eventVersion": "v1"}
	got := amqpTableToMap(table)
	if got["x-correlation-id"] != "abc" || got["eventVersion"] != "v1" {
		t.Errorf("amqpTableToMap = %v, want header values preserved", got)
	}
}
