package messaging

import (
	"context"
	"errors"
	"strings"
	"testing"

	"smart-wardrobe-be/shared/application/constants/eventconstants"
	"smart-wardrobe-be/shared/application/event"

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
//
// 024 T152: this used to assert only that some log contained "Handler error", which
// was the message the framework emitted for EVERY failure - including the ones being
// dead-lettered immediately. So the log actively contradicted what happened to the
// message. All three outcomes are now asserted separately, because the log's whole
// value is that it is the record of the transport decision.
func TestRegisterTypedHandler_LogsHandlerError(t *testing.T) {
	cases := []struct {
		name        string
		handlerErr  error
		wantMessage string
	}{
		{
			name:        "unclassified error is dead-lettered",
			handlerErr:  errors.New("boom"),
			wantMessage: "Unclassified handler error",
		},
		{
			name:        "permanent error is dead-lettered",
			handlerErr:  event.PermanentError{Err: errors.New("bad json")},
			wantMessage: "Poison message",
		},
		{
			name:        "retryable error is requeued",
			handlerErr:  event.RetryableError{Err: errors.New("database is offline")},
			wantMessage: "will be requeued",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client := newTestClient(t)
			handlerErr := tc.handlerErr

			err := client.RegisterTypedHandler(eventconstants.QueueWardrobeCleanup, func(ctx context.Context, msg event.Message) error {
				return handlerErr
			}, event.ConsumerOpts{Concurrency: 1})
			if err != nil {
				t.Fatalf("RegisterTypedHandler returned error: %v", err)
			}

			werr := client.consumers[0].handler(event.Message{
				Body:    []byte(`{}`),
				Headers: map[string]any{"x-correlation-id": "c1"},
			})
			// The framework must not swallow or rewrite the error: startConsumerLoop
			// reads the classification off exactly this value.
			if werr == nil {
				t.Fatal("wrapped handler must propagate handler error")
			}

			logs := typedHandlerLogs(client.logger.(*nopLogger).errs)
			if len(logs) != 1 {
				t.Fatalf("mỗi lỗi phải sinh đúng một log phân loại, có %d: %v", len(logs), logs)
			}
			if !strings.Contains(logs[0], tc.wantMessage) {
				t.Errorf("cần log chứa %q, nhận: %q", tc.wantMessage, logs[0])
			}
			// The line must not claim two different outcomes at once, which is the
			// exact contradiction this task removed.
			if strings.Contains(logs[0], "will be requeued") && !strings.Contains(tc.wantMessage, "requeued") {
				t.Error("log nói sẽ requeue trong khi phân loại là dead-letter")
			}
		})
	}
}

// typedHandlerLogs keeps only the classification lines, dropping the registration
// noise the test client produces without a live broker.
//
// It matches on the message text alone because nopLogger discards zap fields, so the
// correlation id cannot be asserted from here; that it is attached is covered by the
// production code path that reads it back off the message.
func typedHandlerLogs(all []string) []string {
	var out []string
	for _, m := range all {
		if strings.Contains(m, "[TypedHandler]") {
			out = append(out, m)
		}
	}
	return out
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
