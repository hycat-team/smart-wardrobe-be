# Contract: Consumer Framework (`RegisterTypedConsumer[T]`)

**Feature**: `016-rabbitmq-messaging-refactor`

## Purpose

Thay thế 8 wrapper `IWardrobeFashionAnalyzedEventConsumer`... lặp unmarshal+filter bằng 1 generic helper thin.

## Interface

```go
// internal/shared/infrastructure/messaging/consumer_framework.go
// (taxonomy types re-exported here; worker nhận *RabbitMQClient qua DI)

package messaging

type PermanentError struct{ Err error }
func (e PermanentError) Error() string { return e.Err.Error() }
func (e PermanentError) Unwrap() error { return e.Err }

type RetryableError struct{ Err error }
func (e RetryableError) Error() string { return e.Err.Error() }

type ConsumerOpts struct {
    Prefetch    int // 0 = đọc từ Topology theo queue (fallback 20)
    Concurrency int // 0 = đọc từ Topology theo queue (fallback 1)
}

type TypedHandler[T any] func(ctx context.Context, payload T) error

// Shared messaging provides:
func RegisterTypedConsumer[T any](r *RabbitMQClient, queue string, handler TypedHandler[T], opts ConsumerOpts) error
func CorrelationIdFromContext(ctx context.Context) string
```

> **Amend 2026-08-30 (T052)**: bỏ `DeliveryLimit` khỏi `ConsumerOpts` — đó là
> **queue argument** (`x-delivery-limit` trong `queueArgs`, `topology.go`), không
> phải consumer option; helper không thể enforce nó lúc runtime. Defaults
> Prefetch/Concurrency đọc từ registry `Topology` (single source of truth,
> FR-008/FR-022). Helper tái dùng `RegisterConsumer` registry nên
> `reRegisterConsumers` trên reconnect tự hoạt động; per-message ctx derive từ
> `baseCtx` của client (cancel khi `Close()`).

## Behavior

1. `RegisterTypedConsumer` mở **channel riêng** (hoặc từ pool), `Qos(opts.Prefetch,0,false)` per-channel (không global), `Consume(queue, autoAck:false)`.
2. Spawn `opts.Concurrency` goroutines `startConsumerLoop`: `for d := range deliveries { var payload T; if json.Unmarshal(d.Body,&payload)!=nil { Nack(false,false) → DLQ (Permanent) }; ctx:=WithCorrelationId(d.Headers); if err:=handler(ctx,payload); err!=nil { if errors.As(err, PermanentError) { Nack(false,false) } else { Nack(true) or retry TTL } } else { Ack } }`.
3. **Không** `time.Sleep` trong handler; retry do broker. **Không** filter `ItemType` ở app — routing đã phân luồng qua hậu tố `.wardrobe/.brand`.
4. Per-message `context.Context` chứa `correlationId` từ `d.Headers["x-correlation-id"]`.

## Worker Usage (thin)

```go
// internal/modules/wardrobe/presentation/worker/fashion_event_worker.go
func (w *WardrobeFashionEventWorker) Start(ctx context.Context) {
    w.BaseWorker.Start(ctx)
    _ = shared_msg.RegisterTypedConsumer(w.messageClient, shared_msg.QueueWardrobeFashionAnalyzed, w.handle, shared_msg.ConsumerOpts{})
}

// w.handle(ctx, payload) log workerlog + delegate usecase;
// dispatch payload.Status nằm ở usecase HandleFashionEvent (FR-020).
```

## Validation

- `grep -r "time.Sleep" internal/modules/*/infrastructure/messaging/` == 0
- `go test -race` messaging package PASS (không còn single `ch` race)
- Worker file <30 LOC, không unmarshal/filter
