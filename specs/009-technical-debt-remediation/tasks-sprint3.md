# Tasks: Technical Debt & Business Logic Remediation — Sprint 3

**Input**: Design documents from `specs/009-technical-debt-remediation/`

**Scope**: Sprint 3 — Production Hardening (Priority: P2)

| US | Description | Priority |
|----|-------------|----------|
| US1 | HTTP transport fix + RabbitMQ reconnect + outbox pattern | P2 |

---

## Phase 1: Setup (Database Migrations)

- [ ] T001 Create Goose migration for `outbox` table: `CREATE TABLE outbox_records (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), topic VARCHAR(255) NOT NULL, payload JSONB NOT NULL, headers JSONB DEFAULT '{}', status VARCHAR(50) NOT NULL DEFAULT 'pending', created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), published_at TIMESTAMPTZ, attempts INT NOT NULL DEFAULT 0, last_error TEXT)` in `database/migrations/`

---

## Phase 2: User Story 1 — HTTP transport fix + RMQ reconnect + outbox (Priority: P2)

**Goal**: Fix HTTP connection starvation (default transport 2 conn/host). RabbitMQ reconnect tự động re-register consumers. Thêm outbox pattern cho event publishing.

**Independent Test**: HTTP — 100 concurrent requests không bị connection contention. RMQ — restart broker, consumers tự reconnect. Outbox — publish khi RMQ down, event được lưu và publish sau.

### HTTP Transport Fix

- [ ] T002 [P] [US1] Create shared `http.Transport` factory: `internal/shared/infrastructure/httptransport/transport.go` — function `NewTransport() *http.Transport` returning transport with `MaxIdleConns=100`, `MaxIdleConnsPerHost=50`, `MaxConnsPerHost=0`, `IdleConnTimeout=90s`, `DisableCompression=false`
- [ ] T003 [US1] Refactor `internal/shared/infrastructure/ai/core.go` — replace all `&http.Client{Timeout: ...}` with shared transport, each provider keeps its own timeout but shares the transport pool
- [ ] T004 [US1] Refactor `internal/modules/subscription/infrastructure/payment/payos/payos_service_impl.go` — replace `&http.Client{Timeout: 15 * time.Second}` with shared transport, keep timeout

### RabbitMQ Reconnect & Consumer Re-registration

- [ ] T005 [P] [US1] Add consumer registry to `RabbitMQClient` in `internal/shared/infrastructure/messaging/rabbitmq.go` — struct `consumerRegistration{queueName, handler func(amqp.Delivery) error}`; methods `RegisterConsumer(queueName, handler)` and `reRegisterConsumers()`
- [ ] T006 [US1] Modify `RabbitMQClient.handleReconnect()` in `internal/shared/infrastructure/messaging/rabbitmq_connection.go` — after successful reconnect, call `r.reRegisterConsumers()` to re-bind all registered queues and restart consumer goroutines
- [ ] T007 [US1] Update reconnect loop — replace infinite `for { ... time.Sleep(5s) }` with exponential backoff (1s, 2s, 4s, 8s, max 30s), add context cancellation to break loop on shutdown in `internal/shared/infrastructure/messaging/rabbitmq_connection.go`
- [ ] T008 [US1] Update all existing workers (FashionInitWorker, FashionAnalyzeWorker, SearchSyncWorker) to register their consumers via `RegisterConsumer` instead of calling `Consume` directly — files in `internal/modules/*/presentation/worker/`

### Outbox Pattern

- [ ] T009 [P] [US1] Create outbox entity: `internal/modules/shared/infrastructure/messaging/outbox.go` — define `OutboxRecord` struct matching migration table, `OutboxRepository` interface with `Create(ctx, record)`, `ClaimPending(ctx, limit)`, `MarkPublished(ctx, id)`, `MarkFailed(ctx, id, err)`
- [ ] T010 [US1] Implement outbox repository: `internal/modules/shared/infrastructure/messaging/outbox_repo_impl.go` — GORM implementation using shared db connection, `ClaimPending` uses `SELECT ... FOR UPDATE SKIP LOCKED`
- [ ] T011 [P] [US1] Refactor `RabbitMQClient.Publish()` in `internal/shared/infrastructure/messaging/rabbitmq_pubsub.go` — ALWAYS write `OutboxRecord` as primary path (transactional outbox). Outbox worker publishes to RMQ asynchronously. Remove direct RMQ publish path. This guarantees at-least-once delivery per constitution Principle III.
- [ ] T012 [P] [US1] Create outbox worker: `internal/modules/shared/presentation/worker/outbox_worker.go` — periodic worker (ticker 1s), calls `ClaimPending` → publish to RMQ → `MarkPublished` on success
- [ ] T013 [US1] Register outbox worker in `internal/bootstrap/app.go` — add `OutboxWorker` to `AppWorkers` struct, start in `Run()`, stop in shutdown sequence

**Checkpoint**: US1 hoàn chỉnh — HTTP transport fixed, RMQ reconnect + consumer re-registration works, outbox pattern ensures at-least-once delivery

---

## Dependencies & Execution Order

### Phase Dependencies

```
T001 ───── T002 ───── T003 ───── T004 (HTTP transport)
      │
      └──── T005 ───── T006 ───── T007 ───── T008 (RMQ reconnect)
      │
      └──── T009 ───── T010 ───── T011 ───── T012 ───── T013 (Outbox)
```

### Parallel Opportunities

| Tasks | Reason |
|-------|--------|
| T002, T005, T009 | Khác file, không phụ thuộc — tạo transport, consumer registry, outbox entity song song |

### Implementation Strategy

1. **Phase 1**: T001 (migration) — tạo bảng outbox
2. **HTTP transport** (T002-T004): Độc lập, ít rủi ro
3. **RMQ reconnect** (T005-T008): Cần test kỹ với restart RMQ thực tế
4. **Outbox** (T009-T013): Phụ thuộc T001 (bảng), có thể chạy song song với RMQ reconnect
