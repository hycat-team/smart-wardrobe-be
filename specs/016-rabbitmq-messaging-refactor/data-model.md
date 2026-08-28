# Data Model: Refactor RabbitMQ Messaging

**Feature**: `016-rabbitmq-messaging-refactor` | **Date**: 2026-08-27

## Entities & Attributes

### 1. Exchange (RabbitMQ, không phải DB entity)

| Field | Type | Constraints | Notes |
|-------|------|-------------|-------|
| `name` | string | `smart_wardrobe_exchange` (topic, durable) + `smart_wardrobe_exchange.dlx` (DLX topic, durable) | Giữ 1 exchange + DLX đã chốt |
| `type` | string | `topic` | Không đổi, wildcard `.#` thống nhất |
| `durable` | bool | `true` | `ExchangeDeclare` durable |

### 2. Queue (config `topology.go` + `amqp.Table` args)

| Field | Type | Constraints | Notes |
|-------|------|-------------|-------|
| `queueName` | string | `snake_case` + `_queue` suffix (ví dụ `fashion_analyze_queue`) | 13 queue chính, tồn tại DLQ `*.dlq` |
| `routingKeys` | `[]string` | 1..N keys, format `module.event.action[.wardrobe|.brand]` | Hỗ trợ 1 queue bind nhiều key |
| `exchangeName` | string | `smart_wardrobe_exchange` (chính) hoặc `smart_wardrobe_exchange.dlx` (DLQ) | |
| `args` | `amqp.Table` | `x-queue-type` (`classic`/`quorum`), `x-delivery-limit` (5), `x-dead-letter-exchange`, `x-dead-letter-routing-key`, `x-message-ttl`, `x-max-length`, `x-overflow` | Quorum cho 3 queue quan trọng, DLX cho mọi queue |
| `prefetch` | int | per-queue, default 20, `global=false` | Thay `Qos(1)` cứng |
| `concurrency` | int | per-queue, 1..5, default 1 (3 cho `fashion_analyze`) | Goroutine pool |

**Validation**: `queueName` unique; `routingKeys` must match `eventconstants.Topic*` (+ hậu tố đã chốt); `x-queue-type` phải tại declare time, không đổi qua policy; `durable=true` bắt buộc cho quorum.

### 3. OutboxRecord (`shared.outbox_records` — đã tồn tại)

| Field | Type | Constraints | Notes |
|-------|------|-------------|-------|
| `id` | UUID | PK | `outbox.go:19` |
| `topic` | string `varchar(255)` | routing key, NOT NULL | Sau refactor có hậu tố `.wardrobe/.brand` |
| `payload` | `jsonb` | NOT NULL | DTO event (snake_case internal) |
| `headers` | `jsonb` | `{correlationId, traceparent, eventVersion, publishedAt}` | Trước luôn `{}`, sau inject UUID + v1 |
| `status` | enum | `pending/published/failed` (`processing` transient trong Tx) | `ClaimToPublish` bọc Tx |
| `attempts` | int | 0..`retry_max` | |
| `published_at` | timestamptz | nullable | |
| `last_error` | text | nullable | |
| `created_at`, `updated_at` | timestamptz | | Indexes `status_created_at`, `status_published_at` đã có |

**State Transitions**: `pending --ClaimToPublish Tx--> processing --PublishDirect Confirm--> published` hoặc `--Publish fail--> failed --retryAfter--> pending` (attempts<max) → `published` hoặc `failed` (attempts>=max) → DLQ via app (không DB DLQ). `DeletePublishedBefore` cleanup `published_at < now-retentionDays`.

### 4. Message Envelope (AMQP Publishing)

| Field | Type | Notes |
|-------|------|-------|
| `ContentType` | `application/json` | Hardcode |
| `DeliveryMode` | `2` (persistent) | Durable |
| `CorrelationId` | UUID | Từ `headers.correlationId` |
| `MessageId` | UUID | `outboxRecord.id` |
| `Headers` | `amqp.Table` | `x-correlation-id, x-event-version=v1, x-traceparent` |
| `Timestamp` | `time.Time` | `publishedAt` |
| `Body` | `[]byte` | `json.Marshal(payload)` |

### 5. DLQ Message (derived, không bảng mới)

| Field | Type | Notes |
|-------|------|-------|
| `originalQueue` | string | Source queue name |
| `x-death` | array | RabbitMQ tự thêm: `queue, reason=rejected, count, time` |
| `originalHeaders` | `amqp.Table` | Giữ `correlationId` để trace |
| `lastError` | string | `PermanentError` message, log metric |

### 6. ConsumerOpts (Go config struct)

```go
type ConsumerOpts struct {
    Prefetch      int           // default 20
    Concurrency   int           // default 1
    DeliveryLimit int           // quorum: 5
    RetryTTL      time.Duration // 30s
    QueueType     string        // "classic" | "quorum"
}
```

**Relationships**:
- `Exchange 1 -- * Queue` (topic bindings)
- `Queue 1 -- 1 DLQ` (`x-dead-letter-exchange` → `queue.dlq`)
- `OutboxRecord.topic -- Queue.routingKeys` (outbox `topic` = routing key)
- `OutboxRecord 1 -- 1 Message Envelope` (publish)
- `ConsumerOpts 1 -- 1 Queue` (per-queue tuning via `config.yaml`)

**Indexes/Constraints**: `shared.outbox_records` đã có `idx_outbox_records_status_created_at(status,created_at)` cho `ClaimToPublish` + `idx_outbox_records_status_published_at` cho cleanup. Không thêm migration cho quorum (queue args tại declare).
