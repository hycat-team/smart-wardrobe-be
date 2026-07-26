# Tasks: Technical Debt & Business Logic Remediation — Sprint 3

**Input**: Design documents from `specs/009-technical-debt-remediation/`

**Scope**: Sprint 3 — Production Hardening (Priority: P2) (19 tasks)

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

- [ ] T009 [P] [US1] Create outbox entity: `internal/shared/infrastructure/messaging/outbox.go` — define `OutboxRecord` struct matching migration table, `OutboxRepository` interface with `Create(ctx, record)`, `ClaimPending(ctx, limit)`, `MarkPublished(ctx, id)`, `MarkFailed(ctx, id, err)`
- [ ] T010 [US1] Implement outbox repository: `internal/shared/infrastructure/messaging/outbox_repo_impl.go` — GORM implementation using shared db connection, `ClaimPending` uses `SELECT ... FOR UPDATE SKIP LOCKED`
- [ ] T011 [P] [US1] Refactor `RabbitMQClient.Publish()` in `internal/shared/infrastructure/messaging/rabbitmq_pubsub.go` — inject `OutboxRepository` vào `RabbitMQClient` (qua constructor). `Publish(ctx, topic, payload)` gọi `outboxRepo.Create(ctx, record)` — nếu `ctx` có transaction (từ `db.GetTx(ctx)`), ghi trong transaction đó; nếu không, tạo transaction riêng. Outbox worker publishes to RMQ asynchronously. Remove ALL direct `ch.PublishWithContext` code paths. This guarantees at-least-once delivery per constitution Principle III.
- [ ] T012 [P] [US1] Create outbox worker: `internal/shared/presentation/worker/outbox_worker.go` — periodic worker (ticker 1s), calls `ClaimPending` → publish to RMQ → `MarkPublished` on success
- [ ] T013 [US1] Register outbox worker in `internal/bootstrap/app.go` — add `OutboxWorker` to `AppWorkers` struct, start in `Run()`, stop in shutdown sequence
- [ ] T014 [US1] Verify: scan `internal/shared/infrastructure/messaging/rabbitmq_pubsub.go` — confirm `Publish()` chỉ ghi outbox, không còn `ch.PublishWithContext`. Confirm `OutboxRepository` được inject vào `RabbitMQClient`.

**Checkpoint**: Outbox infrastructure hoàn chỉnh — mọi publish viết outbox record, outbox worker publish lên RMQ. Tuy nhiên caller chưa được restructure để publish trong UoW — xử lý ở Phase 3.

---

## Phase 3: Event Publishing Compliance — Restructure Callers (Priority: P2)

**Goal**: Tất cả `eventPublisher.Publish(ctx, topic, payload)` phải được gọi bên trong `uow.Execute(func(txCtx) error { ... })` với `txCtx`, để outbox record được ghi atomic với business data. Hoàn thành toàn bộ compliance với constitution Principle III.

**Independent Test**: Scan codebase — không còn publish call nào dùng `ctx` ngoài UoW. Mọi `Publish(txCtx, ...)` đều ở trong `uow.Execute`.

- [ ] T015 [P] [US1] Restructure **wardrobe** module: `internal/modules/wardrobe/application/usecase/wardrobe/item/item_uc_write.go` (5 call sites: Clone, Update, UpdatePriority, DeleteBulk, DeleteLocked) + `internal/modules/wardrobe/application/usecase/wardrobe/catalog/catalog_uc.go` (2 call sites: UpdateFashionItem, DeleteFashionItem) — wrap DB writes + publish trong `uow.Execute(txCtx)`, gọi `Publish(txCtx, ...)`. `BatchUploadWardrobeItems` đã trong UoW, verify không break.
- [ ] T016 [P] [US1] Restructure **fashion** module: `internal/modules/fashion/application/usecase/worker/worker_uc.go` (3 call sites ngoài UoW + 1 goroutine `time.AfterFunc`) + `internal/modules/fashion/application/usecase/worker/init_uc.go` (1 call site) — wrap trong `uow.Execute(txCtx)`. Goroutine pattern: xoá `time.AfterFunc` delay, ghi outbox record ngay (outbox worker handle async publish). Đảm bảo goroutine không còn direct publish.
- [ ] T017 [US1] Restructure **subscription** module: `internal/modules/subscription/application/usecase/purchase/subscription_purchase_uc.go` (1 call site) — move `Publish(ctx, ...)` từ sau `uow.Execute` vào **bên trong** callback, dùng `Publish(txCtx, ...)`. Đảm bảo outbox record atomic với purchase transaction.
- [ ] T018 [US1] Verify `internal/modules/brand/application/usecase/item/usecase.go` — Publish đã trong UoW (line 323), confirm không break.
- [ ] T019 [US1] **Final verify**: scan toàn bộ `internal/modules/*/application/usecase/` — confirm mọi `eventPublisher.Publish(` đều dùng `txCtx` (từ UoW) hoặc `ctx` từ `db.InjectTx()`. Confirm không còn direct `amqp.Channel.PublishWithContext`. Confirm goroutine pattern đã fix.

**Checkpoint**: US1 hoàn chỉnh — HTTP transport fixed, RMQ reconnect + consumer re-registration works, outbox + caller compliance với constitution Principle III.

---

## Dependencies & Execution Order

### Phase Dependencies

```
T001 ───── T002 ───── T003 ───── T004 (HTTP transport)
      │
      └──── T005 ───── T006 ───── T007 ───── T008 (RMQ reconnect)
      │
      └──── T009 ───── T010 ───── T011 ───── T012 ───── T013 (Outbox infra)
                  │                                        │
                  └──── T014 (verify Publish refactor) ─────┤
                                                           │
                    T015 ───── T016 ───── T017 ───── T018 ──┤ (Caller restructuring)
                                                           │
                    └──── T019 (final verify) ──────────────┘
```

### Parallel Opportunities

| Tasks | Reason |
|-------|--------|
| T002, T005, T009 | Khác file, không phụ thuộc — tạo transport, consumer registry, outbox entity song song |
| T015, T016, T017 | Restructure các module khác nhau — wardrobe, fashion, subscription song song |

### Implementation Strategy

1. **Phase 1**: T001 (migration) — tạo bảng outbox
2. **HTTP transport** (T002-T004): Độc lập, ít rủi ro
3. **RMQ reconnect** (T005-T008): Cần test kỹ với restart RMQ thực tế
4. **Outbox infrastructure** (T009-T014): T009-T013 tạo outbox table/repo/worker, T011 refactor Publish() với OutboxRepository inject. T014 verify Publish() đã clean
5. **Caller restructuring** (T015-T019): T015-T018 restructure từng module. T019 verify cuối
