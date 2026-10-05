# Messaging Guide (RabbitMQ)

> Dành cho dev thêm queue/topic, xử lý lỗi, tune hàng đợi. Chi tiết cơ chế xem `docs/system-design/event-driven-design.md`.

## Thêm queue/topic mới (chỉ 1 nơi + 1 worker)

1. Thêm hằng topic vào `internal/shared/application/constants/eventconstants/eventconstants.go` (ví dụ `TopicOutfitCreated = "outfit.event.created"` và hậu tố `.wardrobe/.brand` nếu cần phân luồng).
2. Thêm hằng `QueueOutfitCreated` vào `internal/shared/application/constants/eventconstants/queues.go` (shared vocabulary — cùng chỗ với topic names), và `RoutingKey*` vào `internal/shared/infrastructure/messaging/rabbitmq_constants.go` tham chiếu hằng trên.
3. Thêm `Binding` vào `internal/shared/infrastructure/messaging/topology.go` `Topology` (ví dụ `{Queue: QueueOutfitCreated, RoutingKeys: []string{RoutingKeyOutfitCreated}, Args: queueArgs(...), Prefetch:20, Concurrency:1}`) — không sửa `rabbitmq_connection.go`. Thêm DLQ tương ứng vào `DLQTopology`.
4. Tạo worker thin `internal/modules/<module>/presentation/worker/<name>_worker.go` embed `*worker.BaseWorker`; `Start(ctx)` gọi `event.RegisterTypedConsumer(w.messageClient, eventconstants.QueueX, w.handle, event.ConsumerOpts{})` với `w.handle(ctx, payload T)` delegate usecase. Unmarshal + correlationId ctx do helper xử lý; `ConsumerOpts{}` để Prefetch/Concurrency đọc từ `Topology` (FR-008/FR-022). Worker nhận interface `event.IMessageConsumer` qua DI (wire bind về RabbitMQClient) — presentation không import infrastructure; dùng `w.Context()` (BaseWorker) cho goroutine phụ để graceful shutdown đúng.
5. Wire worker vào `internal/bootstrap/app.go` `AppWorkers` (concrete pointer) + `provider.go`.
6. `make wire && make build`, verify `curl -u guest:guest http://localhost:15672/api/queues | grep <queue>`.

> Lưu ý lịch sử: trước 016-T051..T057, mỗi module có adapter `infrastructure/messaging/consumer.go` + interface `application/interface/event/` bọc `RegisterConsumer` thủ công. Đã xóa toàn bộ (8 wrapper) — mọi consumer giờ đi qua `RegisterTypedConsumer`; KHÔNG tạo lại adapter module cho consume.

## Đổi queue type (classic ↔ quorum)

**Khác thêm queue mới:** thêm queue chỉ cần `topology.go` khai báo thêm. Đổi *type* của một
queue đã tồn tại thì không được — RabbitMQ coi `x-queue-type` là argument bất biến, nên khai
báo quorum lên một queue đang là classic sẽ bị từ chối:

```
406 PRECONDITION_FAILED - inequivalent arg 'x-queue-type' for queue '<tên>' in vhost '/'
```

Khi đó **mọi service khai báo topology đó không khởi động được** (`failed to initialize
account service: ... rabbitmq initialization failed after 3 attempts`). Cách duy nhất là
drain → xoá → khai báo lại (ADR-005 chấp nhận downtime cho việc này).

Vì vậy CI chạy check trước mỗi deploy: `Check RabbitMQ queue topology` trong
`deploy-vps-services.yml` và `deploy-aws.yml`. Đỏ ở bước đó là tín hiệu sớm, trước khi có
container nào crash-loop. Check là read-only; **migrate không bao giờ tự động**.

Thao tác cụ thể (script, lệnh, xử lý message) — xem
[`deployments/scripts/rabbitmq-topology/README.md`](../../deployments/scripts/rabbitmq-topology/README.md).

Trước khi deploy lần đầu lên môi trường mới, các queue mới sẽ ở trạng thái `MISSING` và
không làm đỏ CI (service tự tạo lúc khởi động).

## Phân loại lỗi Permanent vs Retryable

- **Permanent** (`event.PermanentError{Err: err}`): poison JSON, schema mismatch → `Nack(false,false)` → DLX `*.dlq` quorum, không requeue. Unmarshal JSON fail được helper tự trả Permanent — handler không cần bọc.
- **Retryable** (`event.RetryableError{Err: err}`): `elasticsearch is offline`, DB timeout, transient 5xx → `Nack(false,true)` requeue/TTL; vượt `x-delivery-limit=5` tự vào DLQ. Handler error KHÔNG bọc là mặc định requeue (taxonomy trong `startConsumerLoop`).
- Detection offline thật nằm ở usecase: `search_sync_uc.go` `IsOfflineError` (dùng cho log `willRequeue`), không phải string check ở consumer.
- **Không** `time.Sleep` trong handler; retry do broker.

## Replay DLQ

- DLQ là `queue.dlq` quorum bind vào `smart_wardrobe_exchange.dlx`. Kiểm tra depth: `rabbitmqctl list_queues name messages | grep dlq`.
- Replay thủ công: consume từ DLQ, fix payload, `Publish` lại qua outbox với cùng `correlationId`.
- DLQ durable, có retention; không để poison pile vô hạn.

## Tune prefetch / concurrency

- `configs/config.yaml` `rabbitmq.prefetch` (default 20); per-queue source of truth là `topology.go` `Prefetch/Concurrency` — `RegisterTypedConsumer` với `ConsumerOpts{}` tự đọc theo queue name (override nếu cần truyền opts).
- Quorum yêu cầu `global=false`; tăng prefetch để tránh starvation khi ack chảy qua Raft.
- Thay đổi chỉ cần sửa `topology.go` + config, không chạm `shared` khác.

## Checklist PR chạm messaging

- [ ] `topology.go` có binding mới + DLQ tương ứng?
- [ ] `eventconstants` + `rabbitmq_constants` hằng mới tham chiếu symbol, không hardcode string?
- [ ] Worker thin, không chứa business logic/unmarshal, `BaseWorker` đúng, goroutine phụ dùng `w.Context()`?
- [ ] `go vet` + `make wire` pass, `grep -r "time.Sleep" internal/modules/*/infrastructure/messaging/` == 0?
- [ ] `correlationId` propagate: helper inject vào ctx, worker log event.CorrelationIdFromContext(ctx)`?
- [ ] Test poison → DLQ và burst 100 không mất?
- [ ] Nếu **đổi `x-queue-type`** của queue đang tồn tại: đã chạy migrate trên từng môi trường bị ảnh hưởng (VPS + AWS), không chỉ sửa `topology.go`?
