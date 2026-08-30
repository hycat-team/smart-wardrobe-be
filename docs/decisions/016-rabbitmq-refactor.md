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
