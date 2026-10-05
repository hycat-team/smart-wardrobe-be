# Quickstart: Validate Refactor RabbitMQ

**Feature**: `016-rabbitmq-messaging-refactor` | **Spec**: [spec.md](spec.md) | **Research**: [research.md](research.md)

## Prerequisites

- Go 1.25+, `make` (`make wire`, `make build`, `make test`, `make lint`)
- Docker Compose (`deployments/docker-compose.yml` RabbitMQ single-node `3-management-alpine`)
- `RABBITMQ_USER`/`RABBITMQ_PASSWORD` trong `.env` (default `guest/123456`)

## 1. Khởi động infra

```bash
docker compose -f deployments/docker-compose.yml up -d rabbitmq
docker compose -f deployments/docker-compose.yml ps  # healthy
curl -u guest:guest http://localhost:15672/api/queues | jq '.[].name'  # 13 queue chính + DLQ sau refactor
```

## 2. Build & Test

```bash
make wire && make build
go test ./internal/shared/infrastructure/messaging -race -count=1
make test   # toàn bộ test, SC-008
make lint
grep -r "time.Sleep" internal/modules/*/infrastructure/messaging/ || echo "SC-003 PASS: no sleep block"
```

## 3. Validate Topology Registry (SC-001)

```bash
# Thêm binding giả vào internal/shared/infrastructure/messaging/topology.go:
# {Queue: "test_ping_queue", RoutingKeys: []string{"test.event.ping"}, Args: ClassicDLXArgs}
make wire && make build && docker compose -f deployments/docker-compose.yml up -d --build backend
curl -u guest:guest http://localhost:15672/api/queues | grep test_ping_queue  # phải có
# Xóa test binding sau
```

## 4. Validate DLQ (SC-002)

```bash
# Gửi poison JSON vào notification_email_queue
python3 -c "import pika; c=pika.BlockingConnection(pika.ConnectionParameters('localhost')); ch=c.channel(); ch.basic_publish('smart_wardrobe_exchange','notification.email', b'{bad json', properties=pika.BasicProperties(content_type='application/json', delivery_mode=2)); c.close()"
sleep 2
curl -u guest:guest http://localhost:15672/api/queues | jq '.[] | select(.name=="notification_email_queue.dlq") | .messages'  # >=1
```

## 5. Validate Routing hậu tố (SC-004)

```bash
# Publish 2 message wardrobe vs brand, verify mỗi queue chỉ nhận 1
# Dùng outbox: INSERT INTO shared.outbox_records...
# Verify: wardrobe_fashion_analyzed_queue depth 1, brand_fashion_analyzed_queue depth 1, 0 drop metric
```

## 6. Validate Burst Throughput (SC-005)

```bash
# Burst 100 fashion.event.analyze qua outbox (script test)
go run ./scripts/burst_test.go --count 100 --topic fashion.event.analyze
# Đo thời gian pending→Ack cuối, so với baseline Qos(1)+batch50/10s phải ≥3x
```

## 7. Validate Claim race (SC-006)

```bash
go test ./internal/shared/infrastructure/messaging -run TestClaimConcurrent -count=1 -race
```

## 8. Validate Metrics (SC-007)

```bash
curl http://localhost:8080/metrics | grep -E "outbox_pending|queue_depth|dlq_depth|publish_total|reconnect_total"
# hoặc log structured: docker logs backend-closy-dev | grep correlationId
```

## 9. Validate Docs (SC-010)

```bash
grep -q "smart_wardrobe_exchange.dlx" docs/system-design/event-driven-design.md && echo "FR-024 PASS"
grep -q "topology.go" docs/system-design/architecture.md && echo "FR-025 PASS"
ls docs/development/messaging-guide.md docs/decisions/005-rabbitmq-refactor.md && echo "FR-026/027 PASS"
```

## Expected Outcomes

- `make wire && make build && make lint` PASS, `go test -race` PASS
- 13 queue chính + DLQ hiện trên Management UI, không còn 14 khối `if` lặp
- 0 `time.Sleep` block, 0 ACK-drop wardrobe/brand, burst 3x, 0 message mất
- Metrics 6 chỉ số có, docs đúng cơ chế
