# Research: Refactor Hệ thống Messaging RabbitMQ

**Feature**: `016-rabbitmq-messaging-refactor` | **Date**: 2026-08-27 | **Spec**: [spec.md](spec.md) | **Plan**: [plan.md](plan.md)

> Mục tiêu Phase 0: giải quyết mọi `NEEDS CLARIFICATION` + nghiên cứu best practices trước khi thiết kế chi tiết. Tài liệu này đáp ứng yêu cầu user: *nghiên cứu trên internet xem việc nâng lên quorum có khả thi với cấu hình docker compose hiện tại không hay cần sửa đổi*.

---

## 1. Quorum Queues với Docker Compose hiện tại (single-node) — Có khả thi không?

**Decision**: **Khả thi với cấu hình hiện tại (single-node) và sẽ cấu hình sẵn sàng cho quorum nhưng chỉ 1 node trên prod do chưa có kinh phí** (theo yêu cầu mới). Code declare `x-queue-type=quorum` với `durable=true` đã chạy được trên single-node hiện tại (Raft single-member, tolerated failures = 0) để đạt durability + `delivery-limit`/DLX; **không nâng prod lên 3-node cluster trong 016**, chỉ để sẵn ghi chú lộ trình 3-node khi có kinh phí. Spec đã chốt *downtime drain* (không zero-downtime dual-bind) phù hợp single-node quorum — **downtime prod chấp nhận được do chưa release, đang demo**.

**Rationale**:
- **Internet 2025-2026**: `rabbitmq.com/docs/quorum-queues` — Quorum queues dựa trên Raft, yêu cầu khai `x-queue-type=quorum` tại declare time (không đổi qua policy), **works on single-node cluster** với `tolerated failures = 0` (bảng `1 node → 0`, `3 node → 1`, `5 node → 2`) [websearch: quorum-queues, clustering guide]. GitHub Discussion #10876: *“With just one node you will miss on replication but using QQ in single node at first is fine; downside is more resource per QQ but few thousands ok”*.
- **Docker Compose hiện tại** `deployments/docker-compose.yml:62` + `docker-compose.prod.yml:71` đều `rabbitmq:3-management-alpine` single service `rabbitmq` với `volumes: rabbitmq_data`, **không có `RABBITMQ_ERLANG_COOKIE`, không `hostname` cố định, không cluster formation**. Declare `amqp.Table{"x-queue-type":"quorum", "x-delivery-limit":5}` qua `amqp091-go` **không cần sửa compose** — broker 3.12+ chấp nhận quorum trên 1 node (Raft single-member). Test local: `ch.QueueDeclare("test_qq", true, false, false, false, amqp.Table{"x-queue-type":"quorum"})` succeeds.
- **Nhưng HA**: Quorum thực sự cần 3 node odd để chịu 1 node failure và Raft majority. Single-node quorum chỉ cho **durability + delivery-limit + DLX** (poison handling) chứ không chống node loss. Đây là trade-off đã chốt *downtime drain*: migration classic→quorum không thể in-place, phải `drain → delete → redeclare` (docs: queue type must be at declare time, cannot change via policy).
- **Phiên bản**: `3-management-alpine` (3.13.x) vẫn hỗ trợ quorum đầy đủ; `4.0+` bỏ classic mirrored, khuyên quorum. Không cần nâng image ngay, nhưng `docker-compose.prod.yml` nên ghi chú nâng lên `rabbitmq:4-management-alpine` + `rabbitmq_prometheus` plugin cho quorum + metrics.
- **Resource**: Quorum tốn hơn classic (~32 bytes/msg index + Raft log), `mem_limit 512m` vs `watermark 320MiB` hiện lệch; với 13 queue + DLQ (~26 quorum) vẫn <100MB idle, chấp nhận được cho single-node dev; prod 3-node nên tăng `mem_limit 1g` nếu bật quorum toàn bộ.

**Alternatives considered**:
| Alternative | Đánh giá | Lý do không chọn làm mặc định 016 |
|-------------|----------|--------------------------------------|
| Giữ classic queue toàn bộ, không quorum | Đơn giản, không downtime, đã chạy ổn | Bỏ lỡ `delivery-limit` native + poison handling chuẩn của quorum; classic mirrored đã deprecated 3.13, sẽ phải migrate sau |
| Quorum ngay trên single-node cho mọi queue | Đạt FR-004, code sẵn sàng, drain downtime chấp nhận được | Đã chọn làm **code-ready**: declare quorum cho 3 queue quan trọng (`fashion_analyze_queue`, `wardrobe/brand_fashion_analyzed_queue`) với flag per-env; dev mặc định classic để nhẹ, prod bật quorum qua config |
| 3-node cluster Docker Compose ngay trong 016 | HA thực sự, Raft replication | Vượt scope `constitution.md §I` single deployable + tốn sửa `docker-compose.prod.yml` + `RABBITMQ_ERLANG_COOKIE` + `hostname` + volume x3 + `cluster_status` join script; để lại `docs` mẫu cluster, không ép deploy 3-node trong PR này (user chốt downtime drain, không zero-downtime) |
| Stream thay vì quorum | Append-only, replay, throughput cao | Không cần cho workload FIFO nhỏ này; quorum đủ cho retry/DLQ |

**Cần sửa gì với Docker Compose hiện tại?**
- **Không bắt buộc sửa để chạy quorum code**: giữ nguyên 1 service, chỉ thêm `hostname: rabbitmq` (để tránh cookie/hostname drift) và `rabbitmq.conf` `default_queue_type = quorum` optional.
- **Nếu muốn HA thực sự**: tạo `deployments/docker-compose.cluster.yml` mẫu 3 services `rabbitmq1/2/3` cùng `RABBITMQ_ERLANG_COOKIE=cluster-cookie`, `RABBITMQ_NODENAME=rabbit@rabbitmq-X`, `hostname`, `volumes` riêng, `networks: data`, và script `join_cluster`. Prod hiện tại giữ 1 node nhưng docs ghi rõ lộ trình.

---

## 2. Khai báo Quorum + DLX + Delivery-Limit qua `amqp091-go` (Go)

**Decision**: Dùng `amqp.Table` khi `QueueDeclare`: `{"x-queue-type":"quorum", "x-delivery-limit":5, "x-dead-letter-exchange":"smart_wardrobe_exchange.dlx", "x-dead-letter-routing-key":"<queue>.dlq"}` + DLX exchange `topic` durable riêng. Không dùng policy `rabbitmqctl set_policy` trong code; policy để ops `rabbitmq.conf` nếu cần override. `amqp091-go` hỗ trợ đầy đủ, `QueueDeclare` với `durable=true` bắt buộc cho quorum.

**Rationale**:
- Docs `quorum-queues#declaring`: *must provide at declare time, cannot change via policy*. Go example `oneuptime` + `rabbitmq.com` đều `ch.QueueDeclare(queue, true, false, false, false, amqp.Table{"x-queue-type":"quorum"})`. `amqp091-go` `channel.go: QueueDeclare` nhận `Table`.
- `x-delivery-limit` (quorum poison limit) thay cho `x-max-length` classic; kết hợp `dead-letter-exchange` để message vượt limit tự vào DLQ thay vì drop (docs: *will be dropped or dead-lettered if DLX configured*). Classic `x-message-ttl` vẫn dùng được cho retry TTL.
- DLX phải declare trước: `ExchangeDeclare("smart_wardrobe_exchange.dlx","topic",true,false,false,false,nil)` + bind DLQ `queue.dlq` với routing `#.dlq`.

**Alternatives**: Dùng `rabbitmqctl set_policy` với `delivery-limit` + `dead-letter-exchange` qua policy (ops управля) — không chọn vì plan muốn code explicit, policy để prod tuning sau.

---

## 3. Tách kênh Publish/Consume + QoS per-channel + Concurrency

**Decision**: 1 `publisherChannel` bật `Confirm(false)` + `NotifyPublish` (publisher confirms) cho `PublishDirect`; mỗi `RegisterConsumer` mở channel riêng (hoặc pool) với `Qos(prefetch,0,false)` per-consumer (global=false, quorum yêu cầu `global QoS not supported` — `quorum-queues#global-qos`). `concurrency` per-queue (3 cho `fashion_analyze_queue`) bằng N goroutine `startConsumerLoop` trên N channel riêng hoặc 1 channel với prefetch = concurrency*3.

**Rationale**:
- `rabbitmq.go:29` single `ch` share + `Qos(1,0,false)` global bị overwrite — docs `quorum-queues#global-qos`: *global QoS on quorum will return channel error*.
- Best practice Go `amqp091-go`: *Channels are lightweight, meant to be created per-goroutine* (`oneuptime.com go-rabbitmq`). Tách publisher (`Confirm` để `MarkPublished` chỉ khi broker ack) và consumer channel pools tránh race `Ack` (Channel not safe for concurrent use).
- Prefetch 20 per-consumer (config `rabbitmq.qos_prefetch`) thay vì 1 global: quorum khuyến nghị *higher prefetch to avoid starvation while acks flowing*.

**Alternatives**: Giữ single channel + `mu.RWMutex` — đã chứng minh bottleneck + race, loại.

---

## 4. DLQ & Retry Phân loại Permanent vs Retryable

**Decision**: `RegisterTypedConsumer[T]` unmarshal fail → `PermanentError` → `Nack(false,false)` → DLX (`queue.dlq`). Transient (`ES offline`, DB timeout, `apperror` 5xx) → `RetryableError` → `Nack(true)` hoặc publish sang `retry` exchange với `x-message-ttl 30000` + DLX backoff (không `time.Sleep(5s)` trong handler `wardrobe/consumer.go:40`). Vượt `x-delivery-limit` (5) tự DLQ.

**Rationale**: Hiện `consumer.go:40,190` `Sleep 5s` block goroutine; quorum `x-delivery-count` header track redelivery, nên dùng broker TTL/DLX thay vì sleep. Go handler return typed error để framework quyết `Ack/Nack/retry`.

---

## 5. Fix `ClaimToPublish` Race + Outbox Song song

**Decision**: Bọc `SELECT ... FOR UPDATE SKIP LOCKED LIMIT ?` + `UPDATE status='processing'` trong cùng `db.Transaction`, giữ `retryAfter` precision `time.Duration` (không `int(minutes)` truncate). `OutboxWorker` batch 200 + poll 2s + `errgroup` 10 parallel `PublishDirect` (với publisher confirms), `OutboxListener` fallback poll bền vững không bỏ sau 3 lần (`db/listener.go:102`).

**Rationale**: Hiện `outbox_repo_impl.go:40` `Raw Scan` autocommit không giữ lock khi 2 replica; `batch 50/10s` trần 5msg/s. Go `errgroup` + Tx fix đã là pattern chuẩn transactional outbox.

---

## 6. Docs cho Team Dev (không exhaustive list)

**Decision**: Theo yêu cầu user *“không cần ghi đầy đủ queue hay topic, chỉ cần đúng cơ chế”*, `FR-024` sửa thành: `event-driven-design.md` mô tả đúng **cơ chế** (topic + DLX, routing nguyên tắc, outbox, retry/DLQ, tách channel/prefetch, luồng) thay vì liệt kê 13 queue. `architecture.md §X` + `development/messaging-guide.md` + ADR `016` bổ sung quy ước thêm queue.

**Rationale**: Tránh docs phồng và lệch khi thêm queue; team chỉ cần hiểu mechanism để tuân theo.

---

## Tổng hợp cần sửa với Docker Compose hiện tại?

| Mục | Hiện tại | Cần sửa để quorum khả thi? | Ghi chú |
|-----|----------|----------------------------|---------|
| `deployments/docker-compose.yml` (dev) | single `rabbitmq:3-management-alpine`, no hostname/cookie | **Không bắt buộc**; thêm `hostname: rabbitmq` để stable, optional `default_queue_type` | Quorum declare vẫn chạy |
| `deployments/docker-compose.prod.yml` | single node 512m, watermark 320MiB | **Không sửa scale** — giữ 1 node prod do chưa có kinh phí (đã chốt), `x-queue-type=quorum` single-member vẫn chạy, `512m/320MiB` là chấp nhận cho demo; chỉ ghi chú 3-node cần 1g khi có kinh phí | Quorum 1-node không cần 3-node file |
| `deployments/rabbitmq/rabbitmq.conf` | 3 dòng watermark/disk/mgmt | Thêm note `default_queue_type` optional và `rabbitmq_prometheus` (không bắt buộc cho 1-node), giữ 3 dòng hiện tại là đủ | Không bắt buộc cho single-node quorum |

*Kết luận*: Nâng lên quorum **khả thi ngay với compose hiện tại** ở mức code (declare quorum), không cần sửa compose để chạy; chỉ cần sửa compose nếu muốn HA 3-node thực sự.

---

*Phase 0 complete — mọi NEEDS CLARIFICATION đã resolve, sẵn sàng Phase 1 (data-model, contracts, quickstart).*
