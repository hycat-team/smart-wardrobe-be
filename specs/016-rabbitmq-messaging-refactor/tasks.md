# Tasks: Refactor Hệ thống Messaging RabbitMQ Sạch, Bền và Mở rộng

**Input**: Design documents from `specs/016-rabbitmq-messaging-refactor/` (plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md)

**Prerequisites**: plan.md (tech: Go 1.25+, amqp091-go, single-node quorum-ready), spec.md (5 user stories P1-P3)

**Tests**: Chỉ tạo test tasks khi spec yêu cầu rõ (burst/DLQ/metrics) — ở đây có SC-002/003/004/005/006/007 nên có integration test tasks.

**Organization**: Nhóm theo user story để độc lập test/deploy.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Chuẩn bị nhánh, tài liệu, và khung kiểm tra

- [ ] T001 Tạo nhánh `016-rabbitmq-messaging-refactor` và đảm bảo `specs/016-rabbitmq-messaging-refactor/` đầy đủ (plan.md, research.md, data-model.md, contracts/, quickstart.md)
- [ ] T002 [P] Đọc `docs/system-design/design-pattern.md` và `.specify/memory/constitution.md §I-IV` để nắm Clean Architecture + Worker pattern
- [ ] T003 [P] Chụp baseline hiện tại: `rabbitmqctl list_queues name messages consumers type` và `grep -c "DeclareAndBindQueue" internal/shared/infrastructure/messaging/rabbitmq_connection.go` (14) + `grep -r "time.Sleep" internal/modules/*/infrastructure/messaging/` (2 hits)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Hạ tầng chung phải xong trước mọi US; single exchange + DLX, outbox fix

**⚠️ CRITICAL**: Không làm US nào cho tới khi xong phase này

- [ ] T004 Tạo `internal/shared/infrastructure/messaging/topology.go` với `ExchangeName/ExchangeDLXName`, `Binding` struct, `QuorumArgs/ClassicDLXArgs`, `Topology []Binding` (13 queue) và `DLQTopology` theo `contracts/messaging-topology.contract.md`
- [ ] T005 Thêm `smart_wardrobe_exchange.dlx` topic durable vào `internal/shared/infrastructure/messaging/rabbitmq_connection.go` (ExchangeDeclare DLX trước queue) và đổi 14 khối `if` thành loop `for _, b := range Topology`
- [ ] T006 [P] Cập nhật `internal/shared/infrastructure/messaging/rabbitmq_helper.go` để `QueueDeclare` nhận `args amqp.Table` (quorum `x-queue-type`, `x-delivery-limit=5`, `x-dead-letter-exchange`, `x-dead-letter-routing-key`)
- [ ] T007 Sửa `internal/shared/infrastructure/messaging/outbox_repo_impl.go:36-50` — bọc `SELECT ... FOR UPDATE SKIP LOCKED LIMIT ?` + `UPDATE status='processing'` trong cùng `db.Transaction` và giữ `retryAfter` precision `time.Duration`
- [ ] T008 Làm `PublishDirect` private trong `internal/shared/infrastructure/messaging/rabbitmq_pubsub.go` (chỉ `OutboxUseCase` gọi) và thêm `headers` inject `correlationId/traceparent/eventVersion` theo `contracts/outbox-headers.contract.md`
- [ ] T009 [P] Externalize per-queue config trong `internal/config/config.go` + `configs/config.yaml` + `configs/config.production.yaml` (`rabbitmq.queues[].prefetch`, `concurrency`, `deliveryLimit`, `queueType`) và `deployments/rabbitmq/rabbitmq.conf` ghi chú single-node quorum + watermark fix
- [ ] T010 `make wire && make build` pass sau foundational

**Checkpoint**: Foundation ready — registry + DLX + outbox Tx an toàn, có thể song song US

---

## Phase 3: User Story 1 - Developer thêm queue không sửa `shared` (Priority: P1) 🎯 MVP

**Goal**: Thêm 1 queue mới chỉ sửa registry + 1 worker, không chạm `RabbitMQClient.connect()` thủ công

**Independent Test**: Thêm `test.event.ping → test_ping_queue` vào `topology.go`, `make wire && make build`, start app, verify queue xuất hiện trên Management UI và consumer nhận message (spec US1 acceptance 1-3)

- [ ] T011 [P] [US1] Cập nhật `internal/shared/infrastructure/messaging/rabbitmq_constants.go` thống nhất wildcard `.#` (sửa `wardrobe.event.*` → `wardrobe.event.#`) và thêm hằng `Queue*/RoutingKey*` mới nếu cần cho test
- [ ] T012 [US1] Verify 14 binding cũ vẫn đủ trong `topology.go` (đối chiếu `QueueElasticsearchSync`, `QueueFashionInit`, `QueueFashionAnalyze`, `QueueSubscriptionPlanChanged`, `QueueNotificationEmail`, `QueueAuditLog`, `QueueWardrobeFashionAnalyzed`, `QueueBrandFashionAnalyzed`, `QueueWardrobeCleanup`, `QueueWardrobeProcessingExhausted`, `QueueElasticsearchProcessingExhausted`, `QueueWardrobeFashionBind`, `QueueBrandFashionBind`) và `make build` pass
- [ ] T013 [P] [US1] Viết script/manual test thêm `test_ping_queue` vào registry và verify Management API (`curl -u guest:guest http://localhost:15672/api/queues | grep test_ping_queue`) theo `quickstart.md` §3
- [ ] T014 [US1] Cập nhật `docs/system-design/architecture.md §X` ghi chú registry pattern (không liệt kê exhaustive queue) và Wire không cần sửa khi thêm queue

**Checkpoint**: US1 xong — thêm queue chỉ 1 nơi, 14 queue cũ vẫn hoạt động

---

## Phase 4: User Story 2 - Không mất message khi consumer lỗi (Priority: P1)

**Goal**: Mọi `Nack` đều vào DLQ hoặc retry, phân loại `Permanent vs Retryable`, xóa `time.Sleep` block

**Independent Test**: Gửi poison JSON → DLQ `*.dlq` (không requeue), gửi ES offline transient → retry/DLQ sau `delivery-limit` 5 (spec US2 acceptance 1-3, SC-002/003)

- [ ] T015 [P] [US2] Tạo `internal/shared/infrastructure/messaging/consumer_framework.go` với `PermanentError`, `RetryableError`, `ConsumerOpts`, `RegisterTypedConsumer[T]` generic theo `contracts/consumer-framework.contract.md` (channel riêng, `Qos(prefetch,0,false)` per-channel, `global=false`)
- [ ] T016 [US2] Tách `RabbitMQClient` trong `internal/shared/infrastructure/messaging/rabbitmq.go` thành `publisherCh` (Confirm + NotifyPublish) và `consumerChPool`/per-consumer channel, xóa single `ch` share + `mu.RWMutex` contention
- [ ] T017 [US2] Đổi tất cả handlers `wardrobe/infrastructure/messaging/consumer.go` và `brand/infrastructure/messaging/consumer.go` để return `PermanentError` (unmarshal fail) vs `RetryableError` (ES offline/DB timeout) thay vì `return err` chung
- [ ] T018 [US2] Xóa `time.Sleep(5*time.Second)` trong `internal/modules/wardrobe/infrastructure/messaging/consumer.go:40,190` và thay bằng `Nack(true)` hoặc retry TTL qua DLX
- [ ] T019 [US2] Declare DLQ queues `*.dlq` (13 DLQ) trong `topology.go`/`DLQTopology` với `x-queue-type=quorum`, `durable=true`, bind vào `ExchangeDLXName`
- [ ] T020 [US2] Đảm bảo idempotency consumer (check `itemId` đã `completed` thì bỏ qua UPDATE nhưng vẫn SSE nếu channel mở - kế thừa 014) trong `wardrobe/presentation/worker/fashion_event_worker.go` và `brand/...`
- [ ] T021 [P] [US2] Integration test DLQ: gửi poison JSON vào `notification_email_queue` và `fashion_analyze_queue`, assert `*.dlq` depth >=1 (`quickstart.md` §4)

**Checkpoint**: US2 xong — 0 message mất, 0 sleep block, DLQ quan sát được

---

## Phase 5: User Story 3 - Throughput cao khi burst (Priority: P2)

**Goal**: Burst 100 msg publish <30s, consume concurrency 3, `ClaimToPublish` không claim trùng

**Independent Test**: Burst 100 `fashion.event.analyze` qua outbox, đo pending→Ack cuối ≥3x faster vs baseline `Qos1+batch50/10s` (spec US3, SC-005/006)

- [ ] T022 [US3] Tăng `OutboxWorker` trong `internal/shared/presentation/worker/outbox_worker.go` từ `batch 50/10s` lên `batch 200` + `poll 2s` + parallel `errgroup` 10 `PublishDirect` với publisher confirms (theo `data-model.md` ConsumerOpts) và tạo `scripts/burst_test.go` để burst 100 `fashion.event.analyze` (dùng cho NFR-002 benchmark)
- [ ] T023 [US3] Sửa `internal/shared/application/usecase/outbox_uc.go:42` loop tuần tự thành parallel publish + `MarkPublished` chỉ khi confirm ack
- [ ] T024 [US3] Thêm `concurrency` per-queue trong `topology.go` (ví dụ `QueueFashionAnalyze: concurrency 3, prefetch 20`) và cập nhật `RegisterTypedConsumer` spawn N goroutines `startConsumerLoop` trên N channel riêng
- [ ] T025 [P] [US3] Cấu hình `Qos(prefetch=20, global=false)` per-channel trong `internal/shared/infrastructure/messaging/rabbitmq_pubsub.go:99` (thay `Qos(1,0,false)` cứng) và verify không `global QoS` error với quorum
- [ ] T026 [P] [US3] Test `ClaimToPublish` concurrent 2 replica: `go test -run TestClaimConcurrent -race` trong `internal/shared/infrastructure/messaging/` (SC-006)
- [ ] T027 [US3] Sửa `internal/shared/infrastructure/db/listener.go:102` reconnect không bỏ sau 3 lần, fallback poll bền vững

**Checkpoint**: US3 xong — burst 100 <30s, Qos per-channel, claim an toàn

---

## Phase 6: User Story 4 - Quan sát và vận hành (Priority: P2)

**Goal**: Expose 6 metrics `outbox_pending/queue_depth/dlq_depth/publish_total/consume_ack_nack/reconnect_total` + `correlationId` trace

**Independent Test**: `curl /metrics` hoặc log structured chứa 6 chỉ số; trigger 10 DLQ và ngắt RabbitMQ 5s verify metrics tăng (spec US4, SC-007)

- [ ] T028 [US4] Thêm metrics trong `internal/shared/infrastructure/messaging/` (counters `publish_total`, `consume_ack/nack`, `reconnect_total`, gauges `queue_depth`, `dlq_depth`, `outbox_pending`) expose qua `prometheus/client_golang` hoặc structured `zap` fields
- [ ] T029 [US4] Inject `correlationId` từ `OutboxRecord.Headers` vào `amqp.Publishing.Headers["x-correlation-id"]` và propagate `context.WithValue` trong `RegisterTypedConsumer`, log cùng `correlationId` ở publisher và consumer
- [ ] T030 [US4] Wire `BaseWorker.healthFn` cho messaging trong `internal/bootstrap/app.go` và `internal/shared/presentation/worker/base_worker.go:50` để `/metrics`/`/health` đọc `outbox_pending` + `reconnect_total`
- [ ] T031 [P] [US4] Cập nhật `deployments/rabbitmq/rabbitmq.conf` note `rabbitmq_prometheus` plugin và `deployments/docker-compose.prod.yml` — giữ `mem_limit 512m` vs `watermark 320MiB` cho 1-node quorum là chấp nhận (tolerated failures 0), chỉ ghi chú lộ trình 3-node cần 1g khi có kinh phí (theo research.md §1 và plan.md Constraints)

**Checkpoint**: US4 xong — 6 metrics quan sát được, trace end-to-end

---

## Phase 7: User Story 5 - Thống nhất naming và loại bỏ routing lãng phí (Priority: P3)

**Goal**: Wildcard thống nhất `.#`, routing `fashion.event.analyzed.wardrobe/.brand` loại bỏ ACK-drop 50%

**Independent Test**: Gửi 100 wardrobe + 100 brand, mỗi queue nhận đúng 100, 0 drop; `grep` wildcard thống nhất (spec US5, SC-004)

- [ ] T032 [US5] Cập nhật `internal/shared/application/constants/eventconstants/eventconstants.go:4-20` thêm hằng `TopicFashionAnalyzedWardrobe/Brand`, `TopicFashionFailedWardrobe/Brand`, `TopicFashionItemCreatedWardrobe/Brand` (hoặc suffix `.wardrobe/.brand`)
- [ ] T033 [US5] Cập nhật `internal/shared/infrastructure/messaging/rabbitmq_constants.go:26-38` đồng bộ `RoutingKey*` từ `eventconstants` với hậu tố, thống nhất wildcard `.#` (sửa `RoutingKeyElasticsearchSyncWildcard` từ `wardrobe.event.*` → `wardrobe.event.#`)
- [ ] T034 [US5] Sửa publishers `internal/modules/fashion/application/usecase/worker/worker_uc.go:438,532,356` và `init_uc.go:80` để publish với routing key hậu tố `.wardrobe/.brand` theo `ItemType` (thay vì 1 key chung)
- [ ] T035 [US5] Xóa filter `if ItemType!="wardrobe" return nil` trong `internal/modules/wardrobe/infrastructure/messaging/consumer.go:71,157` và `brand/infrastructure/messaging/consumer.go:34,66` (broker đã routing đúng)
- [ ] T036 [US5] Cập nhật `topology.go` bindings cho `wardrobe_fashion_analyzed_queue` chỉ bind `fashion.event.analyzed.wardrobe` + `fashion.event.failed.wardrobe` (tương tự brand), và `wardrobe_fashion_bind_queue`/`brand_fashion_bind_queue`
- [ ] T037 [P] [US5] Integration test routing: burst 100 wardrobe + 100 brand, assert mỗi queue nhận đúng 100, `grep "ItemType" internal/modules/*/infrastructure/messaging/consumer.go` == 0

**Checkpoint**: US5 xong — naming thống nhất, 0 ACK-drop, `make test` pass

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Tài liệu, ADR, kiểm tra tổng thể, dọn deadcode

- [ ] T038 [P] Cập nhật `docs/system-design/event-driven-design.md` mô tả đúng cơ chế (topic+DLX, routing nguyên tắc, outbox, retry/DLQ, tách channel/prefetch) — không liệt kê exhaustive queue (FR-024)
- [ ] T039 [P] Cập nhật `docs/system-design/architecture.md §X` và `docs/system-design/design-pattern.md` với registry, channel tách, quorum single-node note, `RegisterTypedConsumer[T]` (FR-025)
- [ ] T040 [P] Tạo `docs/development/messaging-guide.md` (thêm queue chỉ registry+worker, Permanent vs Retryable, replay DLQ, tune prefetch/concurrency, checklist PR) (FR-026)
- [ ] T041 [P] Tạo `docs/decisions/016-rabbitmq-refactor.md` ADR 3 quyết định đã chốt: 1 exchange+DLX, downtime drain (demo prod chấp nhận), routing hậu tố `.wardrobe/.brand` (FR-027)
- [ ] T042 [P] Cập nhật `docs/system-design/contract-purity-pattern.md` note nếu cần (FashionItemID đã có từ 015, không đụng)
- [ ] T043 Chạy `quickstart.md` full: `make wire && make build && make lint && go test -race ./internal/shared/infrastructure/messaging` và burst 100 test (SC-001..010)
- [ ] T044 [P] Dọn deadcode: xóa `DeclareAndBindQueue` loop cũ comment, xóa import unused, verify `grep -r "PublishDirect" internal/ --include="*.go"` chỉ 1 hit trong `outbox_uc.go`
- [ ] T045 Final review: `make test` 4 luồng chính (wardrobe batch→SSE, brand→SSE, subscription→aiexecutor, notification/audit) đều pass (SC-008)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: Không dependencies — start ngay
- **Foundational (Phase 2)**: Phụ thuộc Setup — BLOCKS mọi US
- **User Stories (Phase 3-7)**: Phụ thuộc Foundational; có thể song song nếu đủ người, hoặc tuần tự P1→P1→P2→P2→P3 (đề xuất tuần tự để giảm rủi ro, như 015 đã chốt)
- **Polish (Phase 8)**: Phụ thuộc mọi US hoàn tất

### User Story Dependencies

- **US1 (P1) Topology Registry**: Sau Foundational, không phụ thuộc US khác — MVP đầu tiên
- **US2 (P1) DLQ**: Sau Foundational, phụ thuộc US1 registry (dùng Topology + DLQTopology) nhưng test độc lập; nên làm sau US1
- **US3 (P2) Throughput**: Sau Foundational + US2 (dùng channel tách đã có từ US2), test burst độc lập
- **US4 (P2) Observability**: Sau Foundational, có thể song song với US3 (metrics không block throughput)
- **US5 (P3) Naming/Routing**: Sau Foundational + US2 (vì sửa routing key hậu tố liên quan DLQ/binding), test độc lập

### Within Each User Story

- Định nghĩa registry/args trước khi đổi publishers/consumers
- `RegisterTypedConsumer` framework trước khi sửa workers
- Xóa `time.Sleep` / filter trước khi test DLQ/routing
- Viết integration test DLQ/routing trước khi xóa code cũ (verify FAIL trước)

### Parallel Opportunities

- Phase 1 T002-T003 song song
- Phase 2 T006, T009 song song
- Phase 3 T011, T013 song song
- Phase 4 T015 và T021 prep song song (framework vs test script)
- Phase 5 T025, T026 song song
- Phase 6 T028, T031 song song
- Phase 7 T037 song song
- Phase 8 T038-T042 docs song song

---

## Parallel Example: User Story 2

```bash
# Làm song song framework và test DLQ:
Task: "T015 Tạo consumer_framework.go RegisterTypedConsumer[T] trong internal/shared/infrastructure/messaging/consumer_framework.go"
Task: "T021 Chuẩn bị integration test DLQ script cho notification_email_queue"

# Sau đó sửa publishers/consumers:
Task: "T017 Đổi handlers wardrobe/brand consumer.go return Permanent vs Retryable"
```

---

## Implementation Strategy

### MVP First (US1 Only)

1. Hoàn tất Phase 1 Setup + Phase 2 Foundational (registry + DLX + outbox Tx)
2. Hoàn tất Phase 3 US1 (thêm queue chỉ 1 nơi)
3. **STOP và VALIDATE**: `curl /api/queues` thấy test_ping_queue, 14 queue cũ vẫn hoạt động, `make build` pass → Demo MVP

### Incremental Delivery

1. Setup + Foundational → Foundation ready
2. US1 Registry → Demo thêm queue sạch
3. US2 DLQ (không mất message) → Demo poison → DLQ
4. US3 Throughput (batch 200 + concurrency 3) → Demo burst 100 <30s
5. US4 Observability (6 metrics + trace) → Demo Grafana/log
6. US5 Naming + routing hậu tố → Demo 0 ACK-drop
7. Polish docs + ADR → Release

### Parallel Team Strategy

Với 2 devs:

- Dev A: US1 → US2 → US5 (registry + DLQ + routing)
- Dev B: US3 → US4 (throughput + observability) song song sau Foundational
- Gặp nhau ở Polish docs

---

## Notes

- [P] = khác file, không dependencies
- [US*] = trace tới user story spec.md
- Mỗi US độc lập testable (acceptance scenarios trong spec)
- Commit sau mỗi task hoặc nhóm logic
- Dừng ở mọi checkpoint để validate US độc lập
- Tránh: task mơ hồ, conflict cùng file, dependencies vòng giữa US
- Quorum: chỉ 1 node prod (chưa kinh phí 3-node), `x-queue-type=quorum` single-member, downtime drain chấp nhận do demo chưa release

---

## Phase 9: Convergence

- [x] T046 [US3] Wire workers to use topology concurrency per-queue via RegisterTypedConsumer opts per FR-008 (partial) — cập nhật `wardrobe/presentation/worker/*`, `brand/presentation/worker/*`, `fashion/presentation/worker/*` để Concurrency từ `topology.go` (3 cho `fashion_analyze_queue`) thay vì 1 hardcode
- [x] T047 Wire PublishDirect visibility guard per FR-018 (partial) — đổi `PublishDirect` thành private `publishDirect` hoặc thêm `//go:lint` check chỉ `OutboxUseCase` được gọi, hiện vẫn exported với headers
- [x] T048 Cập nhật `deployments/rabbitmq/rabbitmq.conf` với comment single-node quorum per FR-023 (partial) — file hiện 3 dòng, cần thêm note `default_queue_type` và `prometheus` lộ trình 3-node
- [x] T049 Expose metrics `GetMetrics()` qua HTTP `/metrics` hoặc structured log per FR-021 (partial) — hiện chỉ `metrics.go` in-memory counters, chưa gắn vào `internal/api/routes` hay `BaseWorker.healthFn`

---

## Phase 10: Convergence

- [x] T050 Tạo interface riêng `OutboxPublisher` unexported và đổi `PublishDirect` thành private `publishDirect` per Constitution §III và FR-018 (partial) — **quyết định 2026-08-28: không cần private quá chặt, chỉ cần đảm bảo nơi khác không dùng hàm đó là đủ**; hiện `PublishDirect` đã có guard comment `MUST only be called by OutboxUseCase` và `grep PublishDirect` chỉ 1 hit trong `outbox_uc.go`, coi như pass — không đổi thành private
