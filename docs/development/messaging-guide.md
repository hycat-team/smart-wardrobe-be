# Messaging Guide (RabbitMQ)

> Dành cho dev thêm queue/topic, xử lý lỗi, tune hàng đợi. Chi tiết cơ chế xem `docs/system-design/event-driven-design.md`.

## Thêm queue/topic mới (chỉ 1 nơi + 1 worker)

1. Thêm hằng topic vào `internal/shared/application/constants/eventconstants/eventconstants.go` (ví dụ `TopicOutfitCreated = "outfit.event.created"` và hậu tố `.wardrobe/.brand` nếu cần phân luồng).
2. Thêm `RoutingKey*` vào `internal/shared/infrastructure/messaging/rabbitmq_constants.go` tham chiếu hằng trên.
3. Thêm `Binding` vào `internal/shared/infrastructure/messaging/topology.go` `Topology` (ví dụ `{Queue: QueueOutfitCreated, RoutingKeys: []string{RoutingKeyOutfitCreated}, Args: queueArgs(...), Prefetch:20, Concurrency:1}`) — không sửa `rabbitmq_connection.go`.
4. Tạo worker thin `internal/modules/<module>/presentation/worker/<name>_worker.go` embed `*worker.BaseWorker`, `Start(ctx)` gọi `messaging.RegisterTypedConsumer[T](queue, handler, opts)`.
5. Wire worker vào `internal/bootstrap/app.go` `AppWorkers` (concrete pointer) + `provider.go`.
6. `make wire && make build`, verify `curl -u guest:guest http://localhost:15672/api/queues | grep <queue>`.

## Phân loại lỗi Permanent vs Retryable

- **Permanent** (`messaging.PermanentError{Err: err}`): poison JSON, schema mismatch → `Nack(false,false)` → DLX `*.dlq` quorum, không requeue.
- **Retryable** (`messaging.RetryableError{Err: err}`): `elasticsearch is offline`, DB timeout, transient 5xx → `Nack(false,true)` requeue/TTL; vượt `x-delivery-limit=5` tự vào DLQ.
- Trong `infrastructure/messaging/consumer.go` `wrappedHandler`: unmarshal fail → Permanent, handler error chứa `is offline` → Retryable, còn lại handler error → Retryable (mặc định).
- **Không** `time.Sleep` trong handler; retry do broker.

## Replay DLQ

- DLQ là `queue.dlq` quorum bind vào `smart_wardrobe_exchange.dlx`. Kiểm tra depth: `rabbitmqctl list_queues name messages | grep dlq`.
- Replay thủ công: consume từ DLQ, fix payload, `Publish` lại qua outbox với cùng `correlationId`.
- DLQ durable, có retention; không để poison pile vô hạn.

## Tune prefetch / concurrency

- `configs/config.yaml` `rabbitmq.prefetch` (default 20) và `consumer_concurrency` (default 1, 3 cho `fashion_analyze`); `topology.go` `Prefetch/Concurrency` per-queue override.
- Quorum yêu cầu `global=false`; tăng prefetch để tránh starvation khi ack chảy qua Raft.
- Thay đổi chỉ cần sửa `topology.go` + config, không chạm `shared` khác.

## Checklist PR chạm messaging

- [ ] `topology.go` có binding mới + DLQ tương ứng?
- [ ] `eventconstants` + `rabbitmq_constants` hằng mới tham chiếu symbol, không hardcode string?
- [ ] Worker thin, không chứa business logic, `BaseWorker` đúng?
- [ ] `go vet` + `make wire` pass, `grep -r "time.Sleep" internal/modules/*/infrastructure/messaging/` == 0?
- [ ] `correlationId` propagate trong `OutboxRecord.Headers` và log consumer?
- [ ] Test poison → DLQ và burst 100 không mất?
