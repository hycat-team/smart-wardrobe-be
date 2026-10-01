# ADR 016 - Refactor Hệ thống Messaging RabbitMQ Sạch, Bền và Mở rộng

*   **Status**: Accepted
*   **Decision Date**: 2026-08-27
*   **Decision Maker**: team smart-wardrobe-be (spec 016)

## 1. Context

Hệ thống RabbitMQ trước 016 loạn do làm tới đâu sửa tới đó: 1 exchange `smart_wardrobe_exchange` topic + 13 queue + 14 binding hardcode trong `rabbitmq_connection.go`, single `*amqp.Channel` share + `Qos(1)` global, không DLQ (poison/transient đều `Nack(false,false)` drop), filter `ItemType` ở app gây ACK-drop 50%, `ClaimToPublish` không bọc transaction, `PublishDirect` public dễ bypass outbox, wildcard lẫn `*` vs `.#`. Cần refactor để sạch (thêm queue 1 nơi), bền (không mất message), mở rộng (prefetch/concurrency tunable), single-node prod do chưa có kinh phí và downtime demo chấp nhận được.

## 2. Decision

1.  **Giữ 1 exchange `smart_wardrobe_exchange` + DLX `smart_wardrobe_exchange.dlx`** (không tách per-domain). Registry tập trung `topology.go` (`Binding` + `queueArgs` với `x-dead-letter-exchange`, `x-queue-type=quorum` cho 3 queue quan trọng, `x-delivery-limit=5`), `DLQTopology` 13 DLQ `quorum` bind vào DLX. Prod vẫn 1 node, quorum single-member (tolerated failures 0), không 3-node cluster; `512m/320MiB` giữ nguyên, note 3-node cần 1g khi có kinh phí.

2.  **Quorum migrate chấp nhận downtime drain** (không zero-downtime dual-bind). Vì prod chưa release đang demo, drain → delete → redeclare `x-queue-type=quorum` chấp nhận được; `classic→quorum` không convert in-place. Code declare `quorum` đã chạy trên single-node hiện tại (research internet xác nhận works on 1 node).

3.  **Đổi routing hậu tố `.wardrobe/.brand`** cho `fashion.event.analyzed/failed` và `fashion.item.created` để broker routing đúng, xóa filter `ItemType` ở `wardrobe/brand` consumers (loại bỏ ACK-drop 50%). Wildcard thống nhất `.#`. Tách `publisherChannel` (Confirm) + consumer channel riêng `Qos(prefetch=20, global=false)`, `ConsumerOpts` per-queue, `RegisterTypedConsumer[T]` generic, `PermanentError` vs `RetryableError` (sleep xóa), `correlationId` trong `OutboxRecord.Headers`.

4.  **De-scope `traceparent` header** (Decision Date: 2026-08-30). `OutboxRecord.Headers` chỉ mang `correlationId/eventVersion/publishedAt`, không inject W3C `traceparent`. Lý do: hệ thống là modular monolith — publisher và 13 consumer chạy trong cùng 1 process, `correlationId` (đã propagate qua context per-message và log 2 đầu) đáp ứng đủ nhu cầu trace; `traceparent` chỉ phát huy giá trị khi message đi qua ≥2 process khác nhau với OpenTelemetry SDK + tracing backend (Jaeger/Tempo) — làm bây giờ là xây hạ tầng cho nhu cầu chưa tồn tại. Headers là additive nên bổ sung sau không phá gì. **Reconsider khi**: lần đầu có message đi qua 2 process khác nhau (worker tách deploy, service thứ hai consume cùng broker). Chi tiết: `specs/016-rabbitmq-messaging-refactor/contracts/outbox-headers.contract.md` (task T059).

## 3. Consequences

- **Positive**: Thêm queue chỉ sửa `topology.go` + 1 worker; 0 message mất (poison→DLQ, transient→requeue, `delivery-limit`); burst 100 ≥3x nhanh hơn; 6 cơ chế mô tả đúng trong docs không exhaustive; single-node quorum-ready, dễ nâng 3-node sau.
- **Negative**: Downtime ngắn khi migrate quorum (đã chấp nhận do demo); 13 DLQ chiếm thêm disk/mem (~26 quorum idle <100MB); routing suffix là breaking cho message cũ không hậu tố (đã chốt không dual-bind cũ/mới).
- **Follow-up**: `docs/system-design/event-driven-design.md` + `architecture.md §X` + `development/messaging-guide.md` cập nhật đúng 6 cơ chế; metrics 6 chỉ số hiện qua `workerlog`/`zap`, Prometheus optional sau.

## 4. Bổ sung (2026-10-01) — reconcile topology sau khi ADR này đã chạy

Quyết định 2 nói "drain → delete → redeclare" nhưng **không định nghĩa cách phát hiện
drift**, nên `topology.go` và broker trở thành hai nguồn sự thật độc lập. Hệ quả thật:
`subscription_plan_changed_queue` đã là quorum trên VPS nhưng AWS box chưa đồng bộ;
deploy lần sau khai báo quorum lên queue classic và **cả 3 service crash-loop** với
`406 PRECONDITION_FAILED` (2026-10-01). Đọc log từ xa mất thời gian hơn nhiều so với
một CI step đỏ.

Bổ sung:

1.  **Check tự động trước mỗi deploy.** Bước `Check RabbitMQ queue topology` trong
    `deploy-vps-services.yml` + `deploy-aws.yml` so `x-queue-type` trong registry với
    type broker báo qua Management API (AMQP `QueueInspect` **không** trả type — chỉ
    `Name/Messages/Consumers`). Read-only, exit 1 khi drift ⇒ deploy dừng **trước khi**
    container nào start. `MISSING` không tính lỗi (service tự tạo queue lúc khởi động).
2.  **Migrate vẫn thủ công, không tự động** — giữ nguyên Decision 2. Tool
    `shared/tools/rabbitmq-topology/` (`migrate`) chạy drain → spill JSONL → delete →
    redeclare → replay, từ chối khi queue còn consumer. Thao tác + script:
    `deployments/scripts/rabbitmq-topology/`.
3.  **Tool đọc registry trực tiếp** (`messaging.DesiredQueueTypes()`), không giữ bản
    copy danh sách queue — để tool không thể lệch với `topology.go`.
4.  **Replay qua default exchange** (routing key = tên queue), không qua exchange của
    queue: message đã drain mang routing key lúc *publish*, không nhất thiết trùng
    binding key, nên gửi lại qua topic exchange có thể không khớp binding nào và bị
    drop âm thẩm.

**Consequence mới**: đổi `isQuorum` giờ kéo theo 3 việc vận hành (CI báo đỏ → migrate trên
từng môi trường → verify), thay vì 1. Đánh đổi lấy việc lệch bị chặn ở CI thay vì bằng
crash-loop production. **Consequence của consequence**: quyết định này có hiệu lực **trên
cả hai môi trường** — migrate VPS không đủ, phải migrate AWS (và ngược lại).
