# Implementation Plan: Refactor Hệ thống Messaging RabbitMQ Sạch, Bền và Mở rộng

**Branch**: `016-rabbitmq-messaging-refactor` | **Date**: 2026-08-27 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/016-rabbitmq-messaging-refactor/spec.md`

## Summary

Refactor toàn bộ cơ chế RabbitMQ đang loạn (1 exchange `smart_wardrobe_exchange` topic + 13 queue + 14 binding hardcode trong `rabbitmq_connection.go:54-153`, single channel + `Qos(1)` global, không DLQ, filter `ItemType` ở app) thành hệ thống sạch-bền-mở rộng: registry topology tập trung, tách kênh publish/consume per-channel, `prefetch/concurrency` tunable, DLX + retry phân loại `Permanent vs Retryable`, `ClaimToPublish` an toàn transaction, routing hậu tố `.wardrobe/.brand`, outbox `correlationId`. Giữ 1 exchange + DLX (đã chốt), quorum migrate chấp nhận downtime drain, docs chỉ mô tả đúng cơ chế. Tiếp cận: Phase 0 nghiên cứu quorum feasibility với Docker Compose hiện tại → Phase 1 thiết kế data-model/contracts → Phase 2 implement theo registry + quorum-ready + DLQ.

## Technical Context

**Language/Version**: Go 1.25+ (Wire DI, Gin, Zap)

**Primary Dependencies**: `rabbitmq/amqp091-go v1.11.0` (AMQP 0.9.1), `pgx`/`gorm` (outbox), `prometheus/client_golang` (metrics mới), `google/wire`

**Storage**: PostgreSQL `shared.outbox_records` (transactional outbox), RabbitMQ `3-management-alpine` **single-node cả dev và prod** (`deployments/docker-compose*.yml` giữ 1 service `rabbitmq`, **cấu hình sẵn sàng cho quorum nhưng chỉ 1 node trên prod do chưa có kinh phí**), Elasticsearch 8.12 (sync queue)

**Testing**: `go test` + `go test -race` cho `messaging` package, `testcontainers` hoặc `rabbitmq` healthcheck cho integration (burst 100 msg), `make wire && make build && make lint`

**Target Platform**: Linux server Docker Compose (VPS single-host hiện tại), `Backend` Gin single deployable

**Project Type**: Modular monolith backend (Go) - messaging là shared infrastructure

**Performance Goals**: Burst 100 `fashion.event.analyze` publish <30s (vs 200s baseline `batch50/10s`), consume concurrency 3/queue, `Qos prefetch 20` per-channel (vs `1` global), throughput tăng ≥3x (NFR-002); 0 message mất do `Nack` drop (NFR-003)

**Constraints**: At-least-once via `IUnitOfWork.Execute(txCtx)` (`constitution.md §III`) bắt buộc; single exchange + DLX (đã chốt, không tách per-domain); **quorum cấu hình sẵn sàng nhưng prod chỉ 1 node (chưa đầu tư kinh phí cho 3-node cluster)**; quorum migrate downtime drain (không zero-downtime dual-bind); `appendonly` không; docs không liệt kê exhaustive queue (chỉ đúng cơ chế); `512m` mem_limit RabbitMQ prod vs `320MiB` watermark hiện lệch — giữ 1 node, quorum single-member vẫn durable nhưng tolerated failures = 0

**Scale/Scope**: 13 queue chính + N DLQ (`*.dlq`) + outbox 1 bảng; 16 workers `bootstrap/app.go`; burst 100 ảnh B2C; 45 bảng + 1 sequence, `search_path` multi-schema

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- [x] **I. Pragmatic Modular Monolith**: Single Gin deployable, shared infra init 1 lần qua Wire, không tách microservice. Plan giữ `internal/shared/infrastructure/messaging` làm single owner, thêm `topology.go` registry trong shared - PASS.
- [x] **II. Strict Module Boundaries & DTO Contracts**: Cross-module qua `presentation/contract/` DTO only, không import entities/repos. Plan: registry tập trung trong shared nhưng per-module constants vẫn qua `domain/constants` pure; `RegisterTypedConsumer[T]` generic không leak entity - PASS.
- [x] **II. Pure Contract Usecase**: Contract usecase thuần chỉ `*Repo+mapper`. Refactor messaging không đụng `WardrobeContractUseCase/BrandItemContractUseCase` đã thuần ở 015 - PASS.
- [x] **III. Asynchronous Event-Driven + Outbox**: Mọi publish phải qua `outbox_records` trong cùng Tx + `OutboxWorker` async. Plan: `PublishDirect` private chỉ `OutboxUseCase`, fix `ClaimToPublish` bọc Tx, thêm `correlationId` header - PASS (củng cố).
- [x] **IV. Clean Architecture**: `presentation → application → domain ← infrastructure`, handler thin + `WrapHandler`. Workers ở `presentation/worker/` + `BaseWorker`, delegate usecase interface - PASS.
- [x] **V. Transaction & Error**: `IUnitOfWork.Execute` + `apperror` Vietnamese, stack non-prod. Plan giữ `GetTx(ctx)` cho outbox - PASS.
- [x] **VI. AI Pipeline**: Không đổi reservation-first `FOR UPDATE` ledger - out of scope - PASS.
- [x] **VII. Design Patterns Compliance**: Đọc `design-pattern.md` trước code, `sliceutils.IndexBy` cho stitching - PASS.
- [x] **Tech Stack & DB**: Goose migrations cho mọi schema change (quorum args là queue declare, không DB migration; outbox `headers` đã jsonb không cần migration) - PASS.

**Gate Result**: PASS - không có violation cần justify. Không tạo `Complexity Tracking`.

## Project Structure

### Documentation (this feature)

```text
specs/016-rabbitmq-messaging-refactor/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output
│   ├── messaging-topology.contract.md
│   ├── consumer-framework.contract.md
│   └── outbox-headers.contract.md
└── tasks.md             # Phase 2 output (/speckit-tasks - NOT created here)
```

### Source Code (repository root)

```text
internal/
├── shared/
│   ├── application/constants/eventconstants/eventconstants.go # Topic constants + hậu tố .wardrobe/.brand
│   ├── application/event/consumer.go                         # IMessageConsumer + TypedHandler generic mới
│   ├── infrastructure/messaging/
│   │   ├── rabbitmq.go                  # Tách publisherCh / consumerChPool
│   │   ├── rabbitmq_connection.go       # Loop registry thay vì 14 if
│   │   ├── rabbitmq_helper.go           # DeclareAndBindQueue với args
│   │   ├── rabbitmq_constants.go        # Queue/RoutingKey (thống nhất .#)
│   │   ├── topology.go                  # NEW: Binding registry + QuorumArgs + DLX
│   │   ├── consumer_framework.go        # NEW: RegisterTypedConsumer[T] + error taxonomy
│   │   ├── outbox.go / outbox_repo_impl.go # Fix ClaimToPublish Tx + headers correlationId
│   │   └── rabbitmq_pubsub.go           # PublishDirect private + Qos per-channel
│   ├── presentation/worker/
│   │   ├── outbox_worker.go             # batch 200 + poll 2s + errgroup 10
│   │   └── base_worker.go               # healthFn cho messaging
│   └── infrastructure/db/listener.go    # fallback poll bền vững
├── modules/
│   ├── wardrobe/infrastructure/messaging/consumer.go  # Xóa time.Sleep, bỏ ItemType filter
│   ├── wardrobe/presentation/worker/*_worker.go       # Dùng TypedConsumer thin
│   ├── brand/infrastructure/messaging/consumer.go     # Tương tự
│   ├── brand/presentation/worker/*_worker.go
│   ├── fashion/presentation/worker/*_worker.go        # concurrency 3
│   └── ... (aiexecutor, notification, admin workers)
├── config/config.go + configs/config.yaml / config.production.yaml  # per-queue prefetch/concurrency/deliveryLimit
├── deployments/
│   ├── docker-compose.yml               # dev: 1 node, quorum-ready code (classic/quorum per-env flag)
│   ├── docker-compose.prod.yml          # prod: 1 node duy nhất, quorum-ready (x-queue-type quorum single-member, không 3-node cluster do chưa có kinh phí)
│   └── rabbitmq/rabbitmq.conf           # quorum single-node + prometheus + watermark fix (ghi chú lộ trình 3-node khi có kinh phí)
└── docs/
    ├── system-design/event-driven-design.md  # Cập nhật đúng cơ chế (không exhaustive list)
    ├── system-design/architecture.md §X       # Registry + channel tách + quorum note
    ├── system-design/design-pattern.md        # Worker/messaging pattern mới
    ├── development/messaging-guide.md         # NEW: thêm queue/retry/DLQ replay
    └── decisions/016-rabbitmq-refactor.md     # ADR 3 quyết định đã chốt
```

**Structure Decision**: Giữ modular monolith single project, mọi file nằm trong `internal/` + `configs/` + `deployments/` + `docs/` như hiện hữu. Không tạo project mới.

## Complexity Tracking

> Không có violation cần justify (Constitution Check PASS).
