# Đặc tả Tính năng (Feature Specification): Refactor Hệ thống Messaging RabbitMQ Sạch, Bền và Mở rộng

**Feature Branch**: `016-rabbitmq-messaging-refactor`

**Created**: 2026-08-27

**Status**: Draft

**Input**: User description: "lên spec cho việc nghiên cứu và refactor quan trọng này" - nghiên cứu và refactor toàn bộ cơ chế RabbitMQ (exchange, queue, message, event, topic, producer, consumer, mô hình point-to-point vs pub/sub) hiện đang loạn do làm tới đâu sửa tới đó, để hệ thống sạch hơn, dễ maintain và dễ scale.

---

## Clarifications

### Session 2026-08-27

- Q: Có giữ nguyên 1 exchange `smart_wardrobe_exchange` hay tách per-domain exchange? → A: Giữ 1 topic exchange `smart_wardrobe_exchange` + DLX `smart_wardrobe_exchange.dlx` (không tách per-domain trong scope này).
- Q: Khi migrate queue classic sang quorum (cần drain), có chấp nhận downtime ngắn cho hàng đợi không? → A: Chấp nhận downtime ngắn để drain rồi recreate queue với type quorum (cần cửa sổ bảo trì) — chấp nhận được do prod chưa release, đang giai đoạn demo.
- Q: Với topic `fashion.event.analyzed/failed` đang fan-out lãng phí (2 queue nhận cùng key rồi filter `ItemType` ở app), muốn sửa routing key thành `fashion.event.analyzed.{wardrobe,brand}` để broker routing thay vì app filter không? → A: Đổi routing key có hậu tố phân biệt (`fashion.event.analyzed.wardrobe` / `.brand` và `fashion.event.failed.wardrobe` / `.brand`) để broker routing, loại bỏ filter `ItemType` ở app.

### Session 2026-08-27 (Clarify)

- Q: Giữ 1 topic exchange `smart_wardrobe_exchange` + DLX hay tách per-domain exchange? → A: Giữ 1 exchange + DLX
- Q: Khi migrate queue classic sang quorum, chấp nhận downtime ngắn hay zero-downtime? → A: Chấp nhận downtime ngắn để drain rồi recreate (cửa sổ bảo trì) — prod demo chưa release nên downtime chấp nhận được
- Q: Đổi routing key `fashion.event.analyzed/failed` có hậu tố hay giữ fan-out filter app? → A: Đổi routing có hậu tố `wardrobe`/`brand` để broker routing đúng

---

## Bối cảnh & Mục tiêu Refactor

### Hiện trạng (đã audit)

- **1 exchange topic duy nhất** `smart_wardrobe_exchange` (`internal/shared/infrastructure/messaging/rabbitmq_constants.go:6`) phục vụ 13 topic (`eventconstants.go:4-20`) và 13 queue (`rabbitmq_constants.go:9-24`), 14 binding hardcode trong 1 hàm `connect()` (`rabbitmq_connection.go:54-153`).
- **13 topic / 14 binding**: `wardrobe.event.created/updated/deleted`, `fashion.event.init/analyze/analyzed/failed`, `fashion.item.created`, `fashion.event.processing-exhausted`, `subscription.plan.changed`, `notification.email.#`, `admin.audit.log`. Trong đó 5 topic là point-to-point (1 queue), 5 nhóm là pub/sub fan-out (1 routing key → 2 queue).
- **Producer**: `wardrobe`, `fashion`, `brand`, `subscription`, `notification` (qua `OutboxRepository`), `admin` đều qua transactional outbox `rabbitmq_pubsub.go:43` → `OutboxWorker` → `PublishDirect`.
- **Consumer**: 16 workers trong `internal/bootstrap/app.go` + `wardrobe` (6 consumer), `brand` (2), `fashion` (2), `aiexecutor`, `notification`, `admin` mỗi module 1 queue.
- **Nợ chính**: (1) Topology tập trung trong `shared` vi phạm ranh giới module - thêm queue phải sửa `shared`; (2) Single `*amqp.Channel` cho cả publish + consume + `Qos(1)` global → bottleneck ~5msg/s; (3) `fashion.event.analyzed/failed` và `fashion.item.created` publish 1 lần nhưng deliver 2 queue rồi `if ItemType!="wardrobe" return nil` ở `wardrobe/consumer.go:71` lãng phí; (4) Không có DLQ - `QueueDeclare(nil)` `rabbitmq_helper.go:20` + `Nack(false,false)` `rabbitmq.go:104` drop poison/transient (ES offline `consumer.go:40` sleep 5s rồi vẫn drop); (5) Worker boilerplate lặp 8 lần (~50 LOC/worker); (6) `ClaimToPublish FOR UPDATE SKIP LOCKED` không bọc Tx `outbox_repo_impl.go:40` + `batch 50/10s` trần throughput; (7) Naming không nhất quán `wardrobe.event.*` vs `notification.email.#`; config cứng `prefetch/retry`.

### Mục tiêu

1. **Sạch (Maintainable)**: Topology khai báo tập trung dạng registry/config, per-module tự đăng ký binding, thêm queue không chạm `shared`. Thống nhất naming và loại bỏ filter lãng phí ở application.
2. **Bền (Resilient)**: Không mất message - phân loại `Permanent` (poison → DLQ) vs `Retryable` (transient → retry với backoff/TTL), mọi `Nack` đều có đích đến quan sát được.
3. **Mở rộng (Scalable)**: Tách kênh publish/consume, prefetch và concurrency tunable per-queue, outbox claim an toàn khi chạy nhiều replica, hỗ trợ quorum queue.
4. **Quan sát được (Observable)**: Queue depth, publish/consume rate, outbox lag, DLQ depth, reconnect count có thể giám sát; trace `correlationId` xuyên suốt event.

Ràng buộc hiến pháp phải giữ: **Transactional Outbox at-least-once** trong `IUnitOfWork.Execute(txCtx)` (`constitution.md §III`), **Modular Monolith + Wire DI** (`§I`), **Strict Module Boundaries & DTO Contracts** (`§II` + Pure Contract), **Clean Architecture** (`§IV`), **Workers ở `presentation/worker/` + `BaseWorker`**.

---

## User Scenarios & Testing *(Bắt buộc)*

### User Story 1 - Developer thêm mới một loại sự kiện / hàng đợi mà không phải sửa `shared` (Priority: P1)

Là một developer backend, tôi muốn khi cần thêm một topic/queue mới (ví dụ: `outfit.event.created` cho module outfit), tôi chỉ cần khai báo binding ở module đó hoặc trong registry tập trung có cấu trúc, không phải sửa 3-4 nơi rời rạc trong `shared/infrastructure/messaging`, để giảm rủi ro regression và thời gian review.

**Tại sao ưu tiên P1**: Đây là nỗi đau "loạn" lớn nhất hiện tại - mỗi lần thêm queue phải sửa `rabbitmq_connection.go:54-153` (shared) dù queue thuộc `wardrobe/brand`. Fix giúp maintainability ngay lập tức.

**Độc lập kiểm thử**: Thêm một binding giả (`test.event.ping` → `test_ping_queue`) vào registry/config, chạy `make wire && make build`, khởi động app, verify `rabbitmq_connection.go` tự declare/bind queue mới mà không sửa thêm file `shared` nào khác; gửi 1 message test và verify consumer nhận được.

**Acceptance Scenarios**:

1. **Given** registry topology đã tồn tại, **When** developer thêm 1 entry `test.event.ping → test_ping_queue` vào registry, **Then** app khởi động thành công và queue mới xuất hiện trên RabbitMQ Management mà không cần sửa `RabbitMQClient.connect()` thủ công.
2. **Given** 14 binding hiện tại, **When** refactor xong, **Then** toàn bộ 14 binding vẫn được declare/bind đúng như cũ (không sót queue nào: `elasticsearch_sync_queue`, `fashion_init_queue`, `fashion_analyze_queue`, `subscription_plan_changed_queue`, `notification_email_queue`, `audit_log_queue`, `wardrobe_fashion_analyzed_queue`, `brand_fashion_analyzed_queue`, `wardrobe_cleanup_queue`, `wardrobe_processing_exhausted_queue`, `elasticsearch_processing_exhausted_queue`, `wardrobe_fashion_bind_queue`, `brand_fashion_bind_queue`).
3. **Given** một module mới cần queue riêng, **When** module đó đăng ký queue của mình, **Then** không cần import hay sửa `internal/shared/infrastructure/messaging/rabbitmq_constants.go` ở shared (chỉ dùng hằng của module hoặc registry).

---

### User Story 2 - Hệ thống không làm mất message khi consumer gặp lỗi (Priority: P1)

Là một kỹ sư vận hành, tôi muốn mọi message khi consumer xử lý thất bại đều được phân loại rõ: lỗi vĩnh viễn (poison JSON, payload sai schema) thì vào DLQ để quan sát/sửa, lỗi tạm thời (DB/ES offline, timeout) thì được retry có backoff, không bị `Nack(false,false)` drop thầm lặng như hiện tại.

**Tại sao ưu tiên P1**: Hiện tại **0 DLQ**, mọi `Nack` đều drop (`rabbitmq.go:104`), kể cả `consumer.go:40` sleep 5s rồi vẫn drop khi ES offline - đây là rủi ro mất dữ liệu (P0) đã được audit.

**Độc lập kiểm thử**: Gửi 2 loại message lỗi vào cùng queue: (a) JSON sai định dạng, (b) payload hợp lệ nhưng mock ES offline. Verify (a) vào DLQ queue (`*.dlq`) và không requeue, (b) được retry (requeue hoặc retry exchange với TTL) và cuối cùng thành công khi ES hồi phục.

**Acceptance Scenarios**:

1. **Given** một message có JSON không hợp lệ gửi tới `notification_email_queue`, **When** `EmailWorker` unmarshal thất bại, **Then** message được `Nack(false,false)` và route tới DLQ `notification_email_queue.dlq` (không requeue, không mất), và có log/metric ghi nhận `permanent_error`.
2. **Given** `SearchSyncEventConsumer` gặp lỗi `elasticsearch is offline`, **When** handler trả về lỗi tạm thời, **Then** message được `Nack(true)` hoặc publish sang retry queue với TTL 30s (không drop), và được redeliver sau backoff; verify không còn `time.Sleep(5s)` block trong handler `consumer.go:40,190`.
3. **Given** một message retry quá `x-delivery-limit` (ví dụ 5 lần), **When** vẫn thất bại, **Then** message tự động vào DLQ thay vì loop vô hạn.

---

### User Story 3 - Hệ thống xử lý throughput cao khi burst upload (Priority: P2)

Là một người dùng B2C tải lên hàng loạt 100 ảnh, tôi muốn các sự kiện `fashion.event.analyze` và `fashion.event.analyzed` được xử lý song song, không bị nghẽn ở `Qos(1)` global và `outbox batch 50/10s` như hiện tại, để tiến trình SSE cập nhật nhanh và không timeout.

**Tại sao ưu tiên P2**: Hiện tại `Qos(1)` trên single channel (`rabbitmq_pubsub.go:99`) + `outbox batch 50/10s` (`outbox_worker.go:16`) → trần ~5msg/s toàn hệ thống; `FashionAnalyzeWorker` xử lý 20s/ảnh → ~3 msg/phút/queue - không scale được khi burst.

**Độc lập kiểm thử**: Burst 100 message `fashion.event.analyze` vào outbox, đo thời gian từ `Publish` (ghi outbox) tới khi `FashionAnalyzeWorker` Ack xong; verify với prefetch 20 + concurrency 3 thì throughput tăng >5x so với Qos(1) single.

**Acceptance Scenarios**:

1. **Given** burst 100 message trong outbox, **When** `OutboxWorker` chạy với batch 200 + publish song song, **Then** toàn bộ 100 message được publish lên RabbitMQ trong <30s (thay vì >200s hiện tại).
2. **Given** 3 consumer concurrency cho `fashion_analyze_queue`, **When** 10 message đồng thời tới, **Then** 3 message được xử lý song song (không còn head-of-line blocking do 1 goroutine `startConsumerLoop` `rabbitmq.go:100`).
3. **Given** `ClaimToPublish` chạy trên 2 replica backend, **When** cùng claim 1 batch, **Then** không có message nào bị claim trùng (verify `FOR UPDATE SKIP LOCKED` bọc trong transaction).

---

### User Story 4 - Quan sát và vận hành hàng đợi (Priority: P2)

Là một DevOps/SRE, tôi muốn nhìn thấy trên dashboard: độ sâu hàng đợi (queue depth), tỷ lệ publish/consume, outbox lag (số record `pending`), DLQ depth, số lần reconnect, để chủ động scale và cảnh báo trước khi tắc nghẽn.

**Tại sao ưu tiên P2**: Hiện tại chỉ có `workerlog` in-memory + `zap.Info` (`workerlog.go:32`), không có Prometheus/OTel (`spec 009` đã quyết định không thêm metrics - nay cần lại), `BaseWorker.healthFn` chưa từng được set (`base_worker.go:50`), management UI prod không expose.

**Độc lập kiểm thử**: Sau refactor, query `/metrics` hoặc log structured có đủ 5 chỉ số trên; trigger DLQ 10 message và verify DLQ depth metric tăng; ngắt RabbitMQ 5s và verify reconnect count tăng.

**Acceptance Scenarios**:

1. **Given** hệ thống đang chạy, **When** mở endpoint health/metrics, **Then** thấy `outbox_pending_count`, `queue_depth{queue="fashion_analyze_queue"}`, `dlq_depth`, `publish_total`, `consume_ack/nack_total`, `reconnect_total`.
2. **Given** một queue bị tắc (consumer dừng), **When** queue depth > ngưỡng (ví dụ 1000), **Then** có log cảnh báo hoặc metric vượt ngưỡng (không yêu cầu alerting tự động trong scope này, chỉ cần metric quan sát được).
3. **Given** mỗi event publish qua outbox, **When** trace header `correlationId` được inject vào `OutboxRecord.Headers`, **Then** consumer log chứa cùng `correlationId` để nối trace end-to-end.

---

### User Story 5 - Thống nhất naming và loại bỏ routing lãng phí (Priority: P3)

Là một developer mới onboard, tôi muốn naming của exchange/queue/routingKey nhất quán (`snake_case` queue vs `dot.event` topic rõ ràng, wildcard thống nhất `.#` thay vì lẫn `.*` và `.#`), và routing `fashion.event.analyzed` không còn fan-out lãng phí 2 queue rồi filter `ItemType` ở app.

**Tại sao ưu tiên P3**: Hiện tại `RoutingKeyElasticsearchSyncWildcard = wardrobe.event.*` (`rabbitmq_constants.go:27`) dùng `*` trong khi `RoutingKeyNotificationEmail = notification.email.#` (`:36`) dùng `.#`; `wardrobe_fashion_analyzed_queue` + `brand_fashion_analyzed_queue` cùng bind `fashion.event.analyzed` + `fashion.event.failed` rồi `if ItemType!="wardrobe" return nil` (`consumer.go:71,34`) - lãng phí 50% delivery.

**Độc lập kiểm thử**: Đổi routing `fashion.event.analyzed` thành `fashion.event.analyzed.wardrobe`/`brand` (hoặc giữ 1 queue nếu chọn gộp), gửi 2 message `wardrobe` và `brand`, verify mỗi queue chỉ nhận đúng 1 message (không còn ACK-drop).

**Acceptance Scenarios**:

1. **Given** refactor naming xong, **When** đọc `rabbitmq_constants.go`, **Then** mọi wildcard đều dùng `.#` (hoặc đều dùng `.*` - thống nhất), không còn 2 convention lẫn lộn.
2. **Given** 2 message `fashion.event.analyzed` với `ItemType=wardrobe` và `brand`, **When** publish, **Then** `wardrobe_fashion_analyzed_queue` chỉ nhận message wardrobe, `brand_fashion_analyzed_queue` chỉ nhận brand (verify 0 message bị ACK-drop do filter sai loại).
3. **Given** đổi routing key, **When** chạy toàn bộ test cũ `make test`, **Then** không có consumer nào miss message do binding sai.

---

### Edge Cases

- **Poison message loop**: Message JSON sai mãi mãi → phải vào DLQ ngay lần đầu, không retry vô hạn; DLQ cần có `x-message-ttl` hoặc retention để không đầy disk.
- **Broker outage trong lúc publish outbox**: `PublishDirect` thất bại → `outbox_uc.go:49` `MarkFailed` + `attempts++`, retry sau `retry_interval 5m` (`config.yaml:61`); khi broker hồi phục, `handleReconnect` `rabbitmq_connection.go:179` phải re-declare topology và `reRegisterConsumers` mà không leak goroutine cũ.
- **Duplicate delivery (at-least-once)**: Consumer nhận lại cùng `FashionAnalyzedEventPayload` cho `itemId` đã `completed` → phải idempotent (bỏ qua UPDATE DB nếu status đã xong, vẫn phát SSE nếu channel mở - như `014` đã quy định).
- **Outbox claim race**: 2 `OutboxWorker` parallel claim cùng lúc → `FOR UPDATE SKIP LOCKED` phải bọc trong transaction, không autocommit; nếu không sẽ claim trùng và publish duplicate.
- **Channel/connection race**: `Publish` đang giữ `mu.Lock` trong khi `consumeAMQP` giữ `RLock` trên cùng `ch` (`rabbitmq_pubsub.go:15,92`) → sau refactor tách channel riêng thì không còn contention; verify không còn data race `go test -race`.
- **Queue type migration**: Classic → quorum không convert in-place → cần dual-declare hoặc drain trước khi xóa queue cũ; nếu không sẽ lỗi `PRECONDITION_FAILED`.
- **Backward compatibility routing key**: Nếu đổi `fashion.event.analyzed` thành có hậu tố, message cũ (không hậu tố) sẽ không route → cần giữ binding cũ trong giai đoạn chuyển tiếp (dual binding).
- **Elasticsearch offline kéo dài**: Retry liên tục sẽ đầy queue → cần `x-max-length` hoặc `x-overflow: reject-publish` + DLQ để không OOM (`deployments/rabbitmq/rabbitmq.conf:1` `vm_memory_high_watermark 320MiB`).

---

## Requirements *(Bắt buộc)*

### Functional Requirements

#### Nhóm 1: Topology & Registry (P1 - Sạch)

- **FR-001**: Hệ thống PHẢI tập trung khai báo topology RabbitMQ vào một registry/config duy nhất (ví dụ `topology.go` hoặc `config.yaml` section `rabbitmq.queues`), thay vì 14 khối `if DeclareAndBindQueue` lặp trong `rabbitmq_connection.go:54-153` - thêm/xóa queue chỉ sửa registry.
- **FR-002**: Registry PHẢI định nghĩa cho mỗi queue: `queueName`, `routingKeys[]` (hỗ trợ 1 queue bind nhiều key như `wardrobe_fashion_analyzed_queue` bind cả `fashion.event.analyzed` + `fashion.event.failed`), `exchangeName`, `queueArgs` (`x-queue-type`, `x-dead-letter-exchange`, `x-message-ttl`, `x-max-length`).
- **FR-003**: Hệ thống PHẢI khai báo thêm 1 DLX exchange (ví dụ `smart_wardrobe_exchange.dlx` topic, durable) và mỗi queue chính PHẢI có `x-dead-letter-exchange=DLX` + `x-dead-letter-routing-key` tương ứng (ví dụ `wardrobe.fashion_analyzed.dlq`).
- **FR-004**: Hệ thống PHẢI hỗ trợ queue type `quorum` (qua `x-queue-type=quorum`, `x-delivery-limit`) cho các queue cần bền vững (ít nhất `fashion_analyze_queue`, `wardrobe_fashion_analyzed_queue`, `brand_fashion_analyzed_queue`); migration classic→quorum phải có kế hoạch drain/dual-bind.
- **FR-005**: Hệ thống PHẢI thống nhất wildcard convention: toàn bộ routing key wildcard dùng chung 1 dạng (`.#` multi-word hoặc `.*` single-word), không còn lẫn `wardrobe.event.*` (`rabbitmq_constants.go:27`) vs `notification.email.#` (`:36`).

#### Nhóm 2: Client & Channel Model (P1 - Scale)

- **FR-006**: `RabbitMQClient` PHẢI tách kênh publish và consume: 1 `publisherChannel` (bật `Confirm` mode + `NotifyPublish`) cho `PublishDirect`, và mỗi `RegisterConsumer` mở channel riêng (hoặc pool) cho `Consume` - không còn single `ch *amqp.Channel` share (`rabbitmq.go:29`).
- **FR-007**: Mỗi consumer channel PHẢI có `Qos(prefetch=N, global=false)` với `N` configurable per-queue (ví dụ `rabbitmq.qos_prefetch: 20`), thay vì `Qos(1,0,false)` cứng `rabbitmq_pubsub.go:99` overwrite nhau.
- **FR-008**: Hệ thống PHẢI hỗ trợ `concurrency` per-queue (ví dụ `fashion_analyze_queue: concurrency=3`): mỗi queue có N goroutine `startConsumerLoop` song song (worker pool), thay vì 1 goroutine `rabbitmq.go:74` `go startConsumerLoop`.
- **FR-009**: Hệ thống PHẢI fix `ClaimToPublish` (`outbox_repo_impl.go:36-50`): bọc `SELECT ... FOR UPDATE SKIP LOCKED ... LIMIT ?` và `UPDATE status=processing` trong cùng một `Transaction`, không autocommit; `retryAfterMinutes` phải giữ precision (không `int(minutes)` truncate giây).

#### Nhóm 3: Error Handling & DLQ (P1 - Bền)

- **FR-010**: Hệ thống PHẢI phân loại lỗi consumer thành `PermanentError` (poison: JSON unmarshal fail, schema mismatch) vs `RetryableError` (transient: DB/ES timeout, `elasticsearch is offline`): `Permanent` → `Nack(false,false)` → DLQ, `Retryable` → `Nack(true)` hoặc publish sang retry exchange với TTL/backoff.
- **FR-011**: Hệ thống PHẢI xóa `time.Sleep(5s)` block trong handler (`wardrobe/consumer.go:40,190`) - retry phải do broker (TTL/DLX) hoặc worker pool backoff, không block goroutine consume.
- **FR-012**: Mỗi DLQ queue (ví dụ `fashion_analyze_queue.dlq`) PHẢI durable, có retention/TTL và có thể quan sát (queue depth metric), không để poison pile vô hạn.
- **FR-013**: Hệ thống PHẢI đảm bảo idempotency cho consumer redelivery: nếu `itemId` đã ở trạng thái hoàn thành, consumer bỏ qua UPDATE DB nhưng vẫn phát SSE nếu channel mở (kế thừa quy định `014`).

#### Nhóm 4: Routing & Naming (P2 - Sạch)

- **FR-014**: Hệ thống PHẢI sửa routing lãng phí cho `fashion.event.analyzed/failed` và `fashion.item.created`: thay vì 1 routing key fan-out 2 queue rồi filter `ItemType` ở app (`consumer.go:71,34,66,157`), PHẢI dùng routing key có hậu tố phân biệt `fashion.event.analyzed.wardrobe` / `.brand` và `fashion.event.failed.wardrobe` / `.brand` (tương tự `fashion.item.created.wardrobe` / `.brand`) để broker routing đúng, loại bỏ hoàn toàn ACK-drop 50% do filter sai loại.
- **FR-015**: `RoutingKey*` constants (`rabbitmq_constants.go:26-38`) PHẢI được generate hoặc sync từ `eventconstants.Topic*` (`eventconstants.go:4-20`) một cách nhất quán, không hardcode rời rạc; topic versioning (ví dụ `fashion.event.analyzed.v1`) phải được cân nhắc cho forward compatibility.

#### Nhóm 5: Outbox & Headers (P2 - Bền + Quan sát)

- **FR-016**: `OutboxRecord.Headers` (`messaging/outbox.go:21`) PHẢI được inject `correlationId` (UUID), `traceparent` (nếu có OTEL), `eventVersion` (v1), `publishedAt` - thay vì luôn `json.RawMessage({})` (`rabbitmq_pubsub.go:56`); consumer PHẢI log/propagate cùng `correlationId`.
- **FR-017**: `OutboxWorker` (`outbox_worker.go:16,46`) PHẢI hỗ trợ `batch_size` lớn hơn (ví dụ 200) + `pollInterval` ngắn hơn (ví dụ 2s) + publish song song (ví dụ `errgroup` 10) thay vì loop tuần tự `outbox_uc.go:42` 50/10s; `OutboxListener` reconnect (`db/listener.go:102`) phải không bỏ cuộc sau 3 lần mà fallback poll bền vững.
- **FR-018**: `PublishDirect` (`rabbitmq_pubsub.go:14`) PHẢI là private hoặc chỉ `OutboxUseCase` được gọi - không để usecase khác bypass outbox (hiện public, vi phạm `constitution.md §III`).

#### Nhóm 6: Consumer Framework & Boilerplate (P2 - Maintain)

- **FR-019**: Hệ thống PHẢI cung cấp một generic typed consumer helper (ví dụ `RegisterTypedConsumer[T any](queue string, handler func(ctx context.Context, payload T) error, opts ConsumerOpts)`) để thay thế 8 wrapper `IWardrobeFashionAnalyzedEventConsumer`/`IBrandFashionAnalyzedEventConsumer`... lặp `RegisterConsumer` + unmarshal + filter.
- **FR-020**: Mỗi worker `presentation/worker/*_worker.go` PHẢI giữ thin (chỉ `BaseWorker` + `RegisterTypedConsumer` + delegate usecase), không chứa logic unmarshal/filter; logic đó nằm trong shared helper hoặc `infrastructure/messaging` typed consumer.

#### Nhóm 7: Observability & Config (P2)

- **FR-021**: Hệ thống PHẢI expose metrics/health cho messaging: `outbox_pending_count`, `queue_depth`, `dlq_depth`, `publish_total/failed`, `consume_ack/nack_total`, `reconnect_total`, `outbox_lag_seconds` - qua Prometheus `/metrics` hoặc structured log có thể scrape, thay vì chỉ `workerlog` in-memory.
- **FR-022**: `configs/config.yaml` + `config.production.yaml` + `internal/config/config.go` PHẢI externalize per-queue config: `prefetch`, `concurrency`, `retryTTL`, `deliveryLimit`, `maxLength`, `queueType` - không hardcode `Qos(1)`, `sleep 5s`, `reconnect 1s→30s` trong code.
- **FR-023**: `deployments/rabbitmq/rabbitmq.conf` và `docker-compose.prod.yml` PHẢI được cập nhật để hỗ trợ quorum + DLX + `rabbitmq_prometheus` plugin nếu chọn, và `mem_limit` phải tương xứng `vm_memory_high_watermark` (hiện `512m` vs `320MiB` lệch).

#### Nhóm 8: Tài liệu & Quy ước cho Team Dev (P2)

- **FR-024**: Hệ thống PHẢI cập nhật `docs/system-design/event-driven-design.md` để mô tả **đúng và đủ cơ chế RabbitMQ mà hệ thống thực sự dùng** sau refactor (không yêu cầu liệt kê đầy đủ từng queue/topic): loại exchange (`topic` + DLX), nguyên tắc routing (routing key theo `module.event.action` có/không hậu tố phân biệt), phân biệt point-to-point vs pub/sub ở mức cơ chế, transactional outbox, cơ chế retry/DLQ (phân loại `Permanent` vs `Retryable`, `x-delivery-limit`/TTL), channel tách publish/consume và `prefetch`/`concurrency`, và luồng `outbox → exchange → queue → consumer`. Verify bằng checklist 6 mục: (1) exchange type+DLS, (2) routing principle, (3) outbox, (4) DLQ/retry taxonomy, (5) channel/prefetch, (6) flow diagram — đủ 6 mục coi là pass.
- **FR-025**: Hệ thống PHẢI cập nhật `docs/system-design/architecture.md §X Event-Driven Architecture` và `docs/system-design/design-pattern.md` (mục messaging/worker pattern): mô tả registry `topology.go`, tách kênh `publisherChannel`/`consumerChannel`, `Qos(prefetch, global=false)` per-queue, `concurrency` worker pool, `ClaimToPublish` bọc transaction, `RegisterTypedConsumer[T]`, và quy ước naming `module.event.action[.wardrobe|.brand]` + wildcard `.#`.
- **FR-026**: Hệ thống PHẢI bổ sung/cập nhật `docs/development/` (ví dụ `docs/development/messaging-guide.md` hoặc mục trong `local-setup.md`): hướng dẫn dev cách thêm queue/topic mới (chỉ sửa registry + tạo worker, không sửa `shared`), cách phân loại lỗi `Permanent vs Retryable`, cách replay DLQ, cách tune `prefetch/concurrency` per-queue, và checklist review cho PR chạm messaging.
- **FR-027**: Hệ thống PHẢI ghi ADR ngắn trong `docs/decisions/` (ví dụ `005-rabbitmq-refactor.md` theo `docs/decisions/template.md`) cho 3 quyết định đã chốt: giữ 1 exchange+DLX, downtime drain cho quorum, đổi routing có hậu tố - để team sau này hiểu trade-off.

### Non-Functional Requirements

- **NFR-001**: Tương thích ngược - Toàn bộ 13 topic và routing key hiện tại PHẢI tiếp tục hoạt động trong giai đoạn migration (dual binding cũ+mới), không làm gián đoạn luồng `wardrobe → fashion → wardrobe/brand → SSE` đang chạy.
- **NFR-002**: Hiệu năng - Sau refactor, burst 100 message phải được publish và consume nhanh hơn ít nhất 3x so với baseline `Qos(1)+batch50/10s` (đo bằng thời gian từ outbox `pending` tới Ack cuối cùng).
- **NFR-003**: Độ tin cậy - 0 message mất do `Nack` drop sau refactor (mọi `Nack` đều vào DLQ hoặc retry); verify bằng test poison message.
- **NFR-004**: Khả năng bảo trì - Thêm 1 queue mới chỉ cần sửa 1 nơi (registry) + 1 worker file, không sửa `shared/infrastructure/messaging/rabbitmq_connection.go` thủ công.

### Key Entities

- **Exchange**: `smart_wardrobe_exchange` topic durable + `smart_wardrobe_exchange.dlx` DLX topic. Thuộc tính: `name`, `type=topic`, `durable=true`.
- **Queue**: 13 queue chính + N DLQ queues (`*.dlq`) + N retry queues (nếu dùng TTL). Thuộc tính: `name`, `routingKeys[]`, `args` (quorum, DLX, TTL, maxLength), `prefetch`, `concurrency`.
- **Topic / RoutingKey**: Chuỗi `module.event.action` (ví dụ `fashion.event.analyzed.wardrobe`), có thể có version hậu tố `v1`. Là `topic` field trong `outbox_records` (`messaging/outbox.go:20`).
- **OutboxRecord**: `id`, `topic`, `payload` (JSON), `headers` (`correlationId`, `traceparent`, `eventVersion`), `status` (`pending/published/failed`), `attempts`, `published_at`, `last_error`, `created_at`, `updated_at` (`shared.outbox_records`).
- **Message Envelope**: Bọc `payload` + `headers` khi publish: `contentType=application/json`, `deliveryMode=persistent`, `correlationId`, `messageId`, `timestamp`.
- **DLQ Message**: Message gốc + `x-death` header (RabbitMQ tự thêm) + `last_error` để debug/ replay.
- **ConsumerOpts**: `prefetch`, `concurrency`, `retryTTL`, `deliveryLimit` - config per-queue.

---

## Success Criteria *(Bắt buộc)*

### Measurable Outcomes

- **SC-001**: 100% topology binding được khai báo qua registry/config tập trung - thêm 1 queue mới chỉ sửa 1 nơi (verify bằng code review: `rabbitmq_connection.go` không còn 14 khối `if` lặp).
- **SC-002**: 0 message mất do `Nack` drop - mọi `Nack` đều vào DLQ hoặc retry queue (verify bằng test poison: gửi JSON sai vào mỗi queue chính, assert message xuất hiện trong DLQ tương ứng).
- **SC-003**: Không còn `time.Sleep` block trong consumer handler (`consumer.go:40,190` đã xóa) - retry do broker TTL/DLX, không block goroutine (verify `grep -r "time.Sleep" internal/modules/*/infrastructure/messaging/` = 0).
- **SC-004**: Không còn ACK-drop do filter `ItemType` sai loại - mỗi `fashion.event.analyzed` chỉ deliver đúng 1 queue tương ứng (verify burst 100 wardrobe + 100 brand, mỗi queue nhận đúng 100, 0 drop).
- **SC-005**: Throughput burst 100 message tăng ít nhất 3x so với baseline (đo thời gian outbox pending → Ack cuối, với `Qos(1)+batch50/10s` làm baseline).
- **SC-006**: `ClaimToPublish` an toàn khi chạy 2 replica (không claim trùng) - verify bằng test song song 2 worker claim cùng batch.
- **SC-007**: Metrics/health expose đủ 6 chỉ số: `outbox_pending`, `queue_depth`, `dlq_depth`, `publish_total`, `consume_ack/nack`, `reconnect_total` (verify `curl /metrics` hoặc log structured).
- **SC-008**: Toàn bộ 13 topic/queue hiện tại vẫn hoạt động sau refactor - `make test` + manual test 4 luồng chính (wardrobe batch upload → SSE, brand upload → SSE, subscription plan changed → aiexecutor grant, notification email → SMTP, audit log) đều pass.
- **SC-009**: `make wire && make build && make lint` pass, không còn single `ch` share race (`go test -race` cho `messaging` package pass).
- **SC-010**: Tài liệu cho team dev đã cập nhật và được review: `docs/system-design/event-driven-design.md` + `docs/system-design/architecture.md §X` + `docs/development/messaging-guide.md` (hoặc `design-pattern.md`) + ADR `docs/decisions/005-rabbitmq-refactor.md` đều mô tả đúng 6 cơ chế (exchange+DLS, routing principle, outbox, DLQ/retry taxonomy, channel/prefetch, flow) - không yêu cầu liệt kê đầy đủ từng queue/topic, chỉ cần đúng cơ chế - verify bằng checklist 6 mục trong FR-024 + review PR.

---

## Assumptions

- RabbitMQ phiên bản hiện tại `3-management-alpine` (`docker-compose.yml:63`) có thể bật `rabbitmq_prometheus` hoặc nâng lên `4.x` nếu cần quorum/stable - giả định ops cho phép.
- Transactional outbox với `shared.outbox_records` và `OutboxWorker` (`outbox_worker.go:16`) vẫn là cơ chế publish duy nhất - không cho phép publish trực tiếp bypass outbox.
- SSE Manager vẫn thread-safe và được inject độc lập vào Wardrobe/Brand consumer như hiện tại (`fashion_event_uc.go:114`).
- Các client hiện tại tuân thủ SSE JSON contract `014` - refactor messaging không đổi SSE payload.
- Migration classic→quorum chấp nhận downtime ngắn để drain rồi recreate queue với type quorum (cửa sổ bảo trì đã chốt, chấp nhận được do prod chưa release đang demo), không dùng dual-bind zero-downtime trong scope này.
- `WardrobeItemRes` đã có `FashionItemID` sau `015` - caller enrich không còn phụ thuộc `FashionItem` nil.
- Rollback dùng `git revert` - không cần feature flag vì refactor nội bộ, API contract không đổi (đã thống nhất ở `015`).

---

## Phạm vi áp dụng (Out of Scope)

- **Tách thành microservice vật lý**: Vẫn là modular monolith single deployable (`constitution.md §I`), chỉ refactor messaging trong monolith.
- **Thay đổi API endpoint/DTO frontend**: Giữ nguyên 100% API và SSE contract (`014`), chỉ đổi routing key nội bộ (có dual binding).
- **Campaign / Support Ticket / 3D/AR / Global RBAC**: Các out-of-scope đã liệt kê ở `docs/decisions/004-out-of-scope-2026-08.md` - không thêm queue cho các domain đó.
- **Thay đổi AI Pipeline / Cost Control**: Reservation-first, `FOR UPDATE` usage ledger (`constitution.md §VI`) giữ nguyên.
- **Thay đổi Elasticsearch indexing logic**: Chỉ đổi cách message tới `elasticsearch_sync_queue`, không đổi logic index.
- **Alerting tự động (PagerDuty/Slack)**: Chỉ expose metric, alerting là follow-up ops.
- **Schema Registry / Event Versioning hạ tầng nặng**: Chỉ thêm `eventVersion` header đơn giản, không dựng schema registry riêng.

---

## Tài liệu tham khảo

- `internal/shared/infrastructure/messaging/rabbitmq_constants.go` - Exchange/queue/routing key hiện tại
- `internal/shared/infrastructure/messaging/rabbitmq_connection.go` - 14 binding hardcode
- `internal/shared/infrastructure/messaging/rabbitmq_pubsub.go` - `Publish`/`PublishDirect`/`Consume`
- `internal/shared/infrastructure/messaging/rabbitmq.go` - `RabbitMQClient` single channel + `RegisterConsumer`
- `internal/shared/application/constants/eventconstants/eventconstants.go` - 13 topic
- `internal/shared/infrastructure/messaging/outbox*.go` + `internal/shared/presentation/worker/outbox_worker.go` + `internal/shared/application/usecase/outbox_uc.go` - Outbox pattern
- `internal/modules/wardrobe/infrastructure/messaging/consumer.go` + `internal/modules/brand/infrastructure/messaging/consumer.go` - App filter `ItemType`
- `internal/bootstrap/app.go:27-51` - 16 workers lifecycle
- `.specify/memory/constitution.md` v1.8.0 - §I, §II, §III Outbox, §IV Clean Architecture
- `docs/system-design/event-driven-design.md` - Thiết kế event-driven dự kiến vs thực tế
- `docs/system-design/architecture.md` §VII - DB schema & module boundaries
- `specs/014-decouple-fashion-worker-sse/spec.md` - Choreography & idempotency SSE
- `specs/015-contract-purity-refactor/spec.md` - Pure contract & `FashionItemID`

