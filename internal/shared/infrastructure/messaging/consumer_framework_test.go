package messaging

import (
	"context"
	"errors"
	"strings"
	"testing"

	"smart-wardrobe-be/internal/shared/application/constants/eventconstants"
	"smart-wardrobe-be/internal/shared/application/event"

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
		logger:     &nopLogger{},
		baseCtx:    baseCtx,
		baseCancel: cancel,
	}
}

func TestTopologyConcurrencyDefaults(t *testing.T) {
	if got := topologyConcurrency(eventconstants.QueueFashionAnalyze); got != 3 {
		t.Errorf("fashion_analyze concurrency = %d, want 3 (from Topology)", got)
	}
	if got := topologyConcurrency(eventconstants.QueueElasticsearchSync); got != 1 {
		t.Errorf("elasticsearch_sync concurrency = %d, want 1", got)
	}
	if got := topologyConcurrency("unknown_queue"); got != 1 {
		t.Errorf("unknown queue concurrency = %d, want 1 (fallback)", got)
	}
}

func TestTopologyPrefetchDefaults(t *testing.T) {
	if got := topologyPrefetch(eventconstants.QueueFashionAnalyze); got != 20 {
		t.Errorf("prefetch = %d, want 20 (from Topology)", got)
	}
	if got := topologyPrefetch("unknown_queue"); got != 20 {
		t.Errorf("unknown queue prefetch = %d, want 20 (fallback)", got)
	}
}

// RegisterTypedHandler với zero-value opts phải spawn đúng số consumer
// theo Topology, và wrapper phải inject correlationId vào ctx.
func TestRegisterTypedHandler_WrapsAndResolvesDefaults(t *testing.T) {
	client := newTestClient(t)

	var gotCtx context.Context
	wrapper := func(ctx context.Context, msg event.Message) error {
		gotCtx = ctx
		return nil
	}

	if err := client.RegisterTypedHandler(eventconstants.QueueFashionAnalyze, wrapper, event.ConsumerOpts{}); err != nil {
		t.Fatalf("RegisterTypedHandler returned error: %v", err)
	}
	if len(client.consumers) != 3 {
		t.Fatalf("consumers registered = %d, want 3 (Topology concurrency for fashion_analyze)", len(client.consumers))
	}

	msg := event.Message{
		Body:    []byte(`{}`),
		Headers: map[string]any{"x-correlation-id": "corr-abc"},
	}
	if err := client.consumers[0].handler(msg); err != nil {
		t.Fatalf("wrapped handler returned error: %v", err)
	}
	if got := event.CorrelationIdFromContext(gotCtx); got != "corr-abc" {
		t.Errorf("correlationId in ctx = %q, want %q", got, "corr-abc")
	}
	if gotCtx == nil || gotCtx.Err() != nil {
		t.Error("handler ctx must derive from client baseCtx")
	}
}

// Handler error phải được log kèm queue + correlationId, rồi trả nguyên
// lên trên để startConsumerLoop quyết định ack/nack.
func TestRegisterTypedHandler_LogsHandlerError(t *testing.T) {
	client := newTestClient(t)

	err := client.RegisterTypedHandler(eventconstants.QueueWardrobeCleanup, func(ctx context.Context, msg event.Message) error {
		return errors.New("boom")
	}, event.ConsumerOpts{Concurrency: 1})
	if err != nil {
		t.Fatalf("RegisterTypedHandler returned error: %v", err)
	}

	werr := client.consumers[0].handler(event.Message{Body: []byte(`{}`), Headers: map[string]any{"x-correlation-id": "c1"}})
	if werr == nil {
		t.Fatal("wrapped handler must propagate handler error")
	}
	found := false
	for _, m := range client.logger.(*nopLogger).errs {
		if strings.Contains(m, "Handler error") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a handler-error log among %d logs", len(client.logger.(*nopLogger).errs))
	}
}

func TestErrorTaxonomyClassification(t *testing.T) {
	// Mô phỏng switch trong startConsumerLoop:
	// Permanent → Nack(false,false) → DLQ; Retryable → Nack(false,true) → requeue.
	permanent := event.PermanentError{Err: errors.New("bad json")}
	retryable := event.RetryableError{Err: errors.New("elasticsearch is offline")}

	var pe event.PermanentError
	var re event.RetryableError
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
