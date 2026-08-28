# Contract: Outbox Headers & Publish

**Feature**: `016-rabbitmq-messaging-refactor`

## Purpose

`OutboxRecord.Headers` hiện luôn `{}` (`rabbitmq_pubsub.go:56`); sau refactor phải inject trace để nối publish→consume.

## Headers Schema (`jsonb` trong `shared.outbox_records.headers`)

```json
{
  "correlationId": "uuid-v4",
  "traceparent": "00-<traceId>-<spanId>-01", // optional OTel
  "eventVersion": "v1",
  "publishedAt": "2026-08-27T10:00:00Z",
  "producer": "wardrobe"
}
```

## Publish Contract

- `IEventPublisher.Publish(ctx, topic, payload)` (gọi trong `uow.Execute(txCtx)`): `json.Marshal(payload)` → `OutboxRecord{Topic: topic, Payload: bytes, Headers: {correlationId: uuid.NewString(), traceparent: trace.SpanFromContext(ctx), eventVersion:"v1", publishedAt: now}, Status:"pending"}` → `outboxRepo.Create(txCtx, record)` (dùng `db.GetTx(ctx)`). Không gọi `PublishDirect`.
- `OutboxUseCase.PublishBatch`: `ClaimToPublish` bọc `Transaction{ SELECT ... FOR UPDATE SKIP LOCKED LIMIT batchSize FOR UPDATE ...; UPDATE status='processing' RETURNING }` → loop parallel `errgroup` 10: `PublishDirect(record.Topic, record.Payload, record.Headers)` với **publisher confirms** → `MarkPublished` nếu `NotifyPublish` ack, `MarkFailed` nếu nack/error (attempts++).

## AMQP Publishing (`PublishDirect` private)

```go
func (r *RabbitMQClient) publishDirect(ctx context.Context, topic string, body []byte, headers amqp.Table) error {
    // dùng publisherChannel riêng, Confirm mode
    return r.publisherCh.PublishWithContext(ctx, ExchangeName, topic, false, false, amqp.Publishing{
        ContentType: "application/json",
        DeliveryMode: amqp.Persistent,
        CorrelationId: headers["x-correlation-id"].(string),
        MessageId: uuid.NewString(),
        Timestamp: time.Now(),
        Headers: headers,
        Body: body,
    })
}
```

`PublishDirect` phải **private** (chỉ `OutboxUseCase` gọi), không để usecase khác bypass outbox (`constitution.md §III`).

## Validation

- `grep -r "PublishDirect" internal/ --include="*.go"` chỉ 1 hit trong `outbox_uc.go`
- `headers` trong DB không còn `{}` với `SELECT headers FROM outbox_records LIMIT 1`
- Consumer log chứa cùng `correlationId` với publisher log
