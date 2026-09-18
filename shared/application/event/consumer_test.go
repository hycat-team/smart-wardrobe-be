package event

import (
	"context"
	"errors"
	"testing"
)

// fakeConsumer là IMessageConsumer ghi lại handler/opts mà
// RegisterTypedConsumer delegate xuống, mô phỏng phần implementation.
type fakeConsumer struct {
	registered []func(ctx context.Context, msg Message) error
	opts       ConsumerOpts
	queue      string
}

func (f *fakeConsumer) Consume(string) (<-chan Message, error) { return nil, nil }
func (f *fakeConsumer) RegisterConsumer(string, func(msg Message) error) {
}
func (f *fakeConsumer) RegisterTypedHandler(queue string, handler func(ctx context.Context, msg Message) error, opts ConsumerOpts) error {
	f.queue = queue
	f.opts = opts
	f.registered = append(f.registered, handler)
	return nil
}
func (f *fakeConsumer) Close() {}

type payload struct {
	ItemID string `json:"item_id"`
}

func TestRegisterTypedConsumer_UnmarshalFailIsPermanent(t *testing.T) {
	fake := &fakeConsumer{}
	handlerCalled := false

	if err := RegisterTypedConsumer(fake, "some_queue", func(ctx context.Context, p payload) error {
		handlerCalled = true
		return nil
	}, ConsumerOpts{}); err != nil {
		t.Fatalf("RegisterTypedConsumer returned error: %v", err)
	}
	if len(fake.registered) != 1 {
		t.Fatalf("handlers registered = %d, want 1", len(fake.registered))
	}

	werr := fake.registered[0](context.Background(), Message{Body: []byte("{not-json")})
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

func TestRegisterTypedConsumer_PassesTypedPayload(t *testing.T) {
	fake := &fakeConsumer{}
	var got payload

	if err := RegisterTypedConsumer(fake, "some_queue", func(ctx context.Context, p payload) error {
		got = p
		return nil
	}, ConsumerOpts{}); err != nil {
		t.Fatalf("RegisterTypedConsumer returned error: %v", err)
	}

	if err := fake.registered[0](context.Background(), Message{Body: []byte(`{"item_id":"abc"}`)}); err != nil {
		t.Fatalf("wrapper returned error: %v", err)
	}
	if got.ItemID != "abc" {
		t.Errorf("payload = %+v, want item_id abc", got)
	}
}

func TestRegisterTypedConsumer_HandlerErrorPropagates(t *testing.T) {
	fake := &fakeConsumer{}

	if err := RegisterTypedConsumer(fake, "some_queue", func(ctx context.Context, p payload) error {
		return RetryableError{Err: errors.New("elasticsearch is offline")}
	}, ConsumerOpts{}); err != nil {
		t.Fatalf("RegisterTypedConsumer returned error: %v", err)
	}

	werr := fake.registered[0](context.Background(), Message{Body: []byte(`{}`)})
	var re RetryableError
	if !errors.As(werr, &re) {
		t.Fatalf("error is not RetryableError: %v", werr)
	}
}

func TestRegisterTypedConsumer_DelegatesQueueAndOpts(t *testing.T) {
	fake := &fakeConsumer{}
	opts := ConsumerOpts{Prefetch: 42, Concurrency: 2}

	if err := RegisterTypedConsumer[string](fake, "my_queue", func(ctx context.Context, s string) error { return nil }, opts); err != nil {
		t.Fatalf("RegisterTypedConsumer returned error: %v", err)
	}
	if fake.queue != "my_queue" || fake.opts != opts {
		t.Errorf("delegate mismatch: queue=%q opts=%+v", fake.queue, fake.opts)
	}
}
