# Contract: Messaging Topology Registry

**Feature**: `016-rabbitmq-messaging-refactor`

## Purpose

Thay thế 14 khối `if DeclareAndBindQueue` hardcode trong `rabbitmq_connection.go:54-153` bằng registry tập trung, để thêm queue chỉ sửa 1 nơi.

## Registry Structure (`internal/shared/infrastructure/messaging/topology.go`)

```go
package messaging

import amqp "github.com/rabbitmq/amqp091-go"

const (
    ExchangeName    = "smart_wardrobe_exchange"
    ExchangeType    = "topic"
    ExchangeDLXName = "smart_wardrobe_exchange.dlx"
    ExchangeDLXType = "topic"
)

type Binding struct {
    Queue        string
    RoutingKeys  []string      // 1..N, ví dụ {"fashion.event.analyzed.wardrobe"}
    Exchange     string        // default ExchangeName
    Args         amqp.Table    // x-queue-type, x-delivery-limit, x-dead-letter-*
    Prefetch     int           // per-queue
    Concurrency  int           // per-queue
}

var QuorumArgs = amqp.Table{
    "x-queue-type":               "quorum",
    "x-delivery-limit":           5,
    "x-dead-letter-exchange":     ExchangeDLXName,
    "x-dead-letter-routing-key":  "", // fill per queue: queue+".dlq"
}

var Topology = []Binding{
    {Queue: QueueElasticsearchSync, RoutingKeys: []string{RoutingKeyElasticsearchSyncWildcard}, Args: ClassicDLXArgs, Prefetch: 20, Concurrency: 1},
    {Queue: QueueFashionInit, RoutingKeys: []string{RoutingKeyFashionInit}, Args: ClassicDLXArgs, Prefetch: 20, Concurrency: 1},
    {Queue: QueueFashionAnalyze, RoutingKeys: []string{RoutingKeyFashionAnalyze}, Args: QuorumArgs, Prefetch: 20, Concurrency: 3},
    // ... 11 bindings còn lại
}

var DLQTopology = []Binding{
    {Queue: QueueFashionAnalyze + ".dlq", RoutingKeys: []string{QueueFashionAnalyze + ".dlq"}, Exchange: ExchangeDLXName, Args: amqp.Table{"x-queue-type": "quorum"}},
    // ...
}
```

## Contract Rules

- `DeclareAndBindQueue` loop `for _, b := range Topology { ch.QueueDeclare(b.Queue,true,false,false,false,b.Args); ch.QueueBind(b.Queue, rk, b.Exchange) }` cho mỗi `RoutingKey` trong `RoutingKeys`.
- Wildcard thống nhất `.#` (sửa `wardrobe.event.*` → `wardrobe.event.#` nếu chốt `.#`).
- Quorum migrate: declare với `x-queue-type=quorum` yêu cầu queue chưa tồn tại hoặc đã là quorum; nếu classic tồn tại thì phải drain delete rồi declare lại (downtime đã chốt).
- DLX exchange declare trước queue: `ExchangeDeclare(ExchangeDLXName, ExchangeDLXType, true, ...)`.

## Validation

- `make build` sau khi thêm 1 `Binding` giả `test.event.ping → test_ping_queue` phải hiện queue trên Management UI mà không sửa `rabbitmq_connection.go` thủ công.
- `grep -c "DeclareAndBindQueue" internal/shared/infrastructure/messaging/rabbitmq_connection.go` giảm từ 14 calls xuống 1 loop.
