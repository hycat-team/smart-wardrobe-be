# Event-Driven Design

Event-driven architecture giúp xử lý bất đồng bộ các tác vụ AI tốn thời gian, đảm bảo API Gateway luôn phản hồi nhanh và ranh giới module được tôn trọng qua sự kiện thay vì gọi trực tiếp.

## 1. Cơ chế RabbitMQ đang dùng (016)

Hệ thống dùng **1 topic exchange chính `smart_wardrobe_exchange` + 1 DLX `smart_wardrobe_exchange.dlx`** (durable, type `topic`). Mọi sự kiện đều đi qua **transactional outbox** (`shared.outbox_records`) trong cùng `IUnitOfWork.Execute(txCtx)` rồi `OutboxWorker` publish qua `PublishDirect` (publisher confirms) với `correlationId/eventVersion/publishedAt` trong headers.

*   **Exchange & Routing**: Routing key theo `module.event.action` (ví dụ `wardrobe.event.created`, `fashion.event.analyze`, `subscription.plan.changed`); các sự kiện fashion có hậu tố `.wardrobe/.brand` để broker routing đúng, wildcard thống nhất `.#` (ví dụ `wardrobe.event.#`, `notification.email.#`).
*   **Point-to-point vs Pub/Sub**: Topic exchange nhưng 1 routing key → 1 queue là point-to-point (ví dụ `fashion.event.init` → `fashion_init_queue`), 1 routing key → 2 queue là pub/sub fan-out (ví dụ `fashion.event.processing-exhausted` → `wardrobe` + `elasticsearch`).
*   **Outbox**: `topic` trong `outbox_records` chính là routing key; `OutboxWorker` poll `2s` + `LISTEN outbox_channel`, batch `200` parallel 10 với `FOR UPDATE SKIP LOCKED` bọc transaction.
*   **Retry/DLQ**: Consumer phân loại `PermanentError` (poison JSON → `Nack(false,false)` → DLX `*.dlq` quorum) vs `RetryableError` (ES offline/transient → `Nack(true)` requeue/TTL); `x-delivery-limit=5` + `x-dead-letter-exchange` trên mỗi queue chính.
*   **Channel & Prefetch**: `publisherChannel` riêng (Confirm) + mỗi consumer channel riêng với `Qos(prefetch=20, global=false)`, `concurrency` per-queue (3 cho `fashion_analyze`).
*   **Luồng**: `wardrobe/fashion/brand/subscription` → `uow.Execute(txCtx)` ghi DB + `outbox_records` → `pg_notify` → `OutboxWorker` → `smart_wardrobe_exchange` (topic) → `queue` → `consumer` (unmarshal → handler → Ack/Nack) → DLQ nếu vượt limit.

> Chi tiết queue cụ thể xem `internal/shared/infrastructure/messaging/topology.go` (registry), không liệt kê exhaustive ở đây.

## 2. Ví dụ luồng

- User upload ảnh `wardrobe.event.created` → wardrobe lưu item + outbox → elasticsearch sync worker (`wardrobe.event.#`) index.
- `fashion.event.init/analyze.wardrobe` → fashion init/analyze worker xử lý AI, publish `fashion.event.analyzed.wardrobe` → wardrobe consumer cập nhật DB + SSE.
- `subscription.plan.changed` → aiexecutor grant worker.
- `notification.email` / `admin.audit.log` → email/audit workers.

## 3. Lợi ích kiến trúc

- **Tránh timeout**: Client không chờ AI đồng bộ.
- **Bền vững**: Single-node quorum-ready (1 node prod, tolerated failures 0) + DLQ/TTL + delivery-limit, downtime drain chấp nhận do demo chưa release.
- **Mở rộng**: Thêm queue chỉ sửa `topology.go` + 1 worker, không sửa `rabbitmq_connection.go`; prefetch/concurrency tunable qua `config.yaml`.
- **Quan sát**: `correlationId` xuyên suốt headers, metrics `outbox_pending/queue_depth/dlq_depth/publish/ack_nack/reconnect` qua `workerlog`/`zap`.
