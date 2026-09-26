# Implementation Plan: Xử lý kịch bản theo trạng thái phân tích trang phục AI

**Branch**: `023-analyze-status-handling` | **Date**: 2026-09-26 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/023-analyze-status-handling/spec.md`

## Summary

Bổ sung và chuẩn hóa kịch bản xử lý cho từng trạng thái phân tích AI của trang phục (B2C wardrobe + B2B brand), gồm 7 hạng mục (G1–G7):
1. **G1** — Endpoint xác nhận món `needs_review` → usable (`POST /wardrobe-items/{id}/confirm-review`), bắt buộc danh mục hợp lệ.
2. **G2** — Phát và hiển thị realtime trạng thái `processing` (mới) cho từng món.
3. **G3** — Thử lại an toàn: chỉ cho phép khi trạng thái hợp lệ (thất bại/cần rà soát), chống phiên trùng bằng claim nguyên tử (không cooldown/giới hạn số lần).
4. **G4** — Giữ danh mục người dùng đã sửa khi phân tích lại.
5. **G5/G6** — Brand: thêm trạng thái `failed` (+ lý do), chặn đổi khỏi `failed` bằng thao tác thủ công, thêm endpoint retry-analysis.
6. **G7** — Buffer phát lại sự kiện realtime khi client kết nối lại.
7. Tài liệu playbook + tài liệu tích hợp frontend (`frontend-guide.md`).

Điểm kiến trúc then chốt: ở topology AWS (`APP_MODE=http`/`worker` tách tiến trình), sự kiện SSE được phát sinh trong **worker** nhưng client kết nối ở **http**; do đó cần một kênh chia sẻ **Redis Pub/Sub** làm chặng cuối (last-mile), trong khi luồng nghiệp vụ vẫn dùng RabbitMQ + outbox. Ở VPS (`APP_MODE=all`) cả hai nằm cùng tiến trình nhưng vẫn dùng chung một đường để nhất quán.

## Technical Context

**Language/Version**: Go 1.25+ — multi-module Go workspace (`go.work`); service `commerce` sở hữu module `wardrobe`, `fashion`, `brand`.

**Primary Dependencies**: Gin, Google Wire (DI), GORM, pgvector, RabbitMQ (transactional outbox + topology registry), `redis/go-redis/v9`, Goose (migrations), Swag (Swagger).

**Storage**: PostgreSQL (schema `wardrobe`, `fashion`, `brand`; `search_path` resolve), Redis (cache + Pub/Sub mới). RabbitMQ cho event nghiệp vụ.

**Testing**: Go `testing` + mock thủ công; chạy `make test-race`. Swagger regenerate cho endpoint mới.

**Target Platform**: Linux containers. Hai topology:
- VPS: 3 service container, `APP_MODE=all` (http+worker+consumer chung tiến trình).
- AWS: 6 container, `APP_MODE=http` (giữ SSE) + `APP_MODE=worker` (consumer/worker) tách riêng.

**Project Type**: Backend web-service (REST + SSE + event-driven workers).

**Performance Goals**: Sự kiện realtime tới client gần thời gian thực; buffer phát lại mặc định 100 event/phiên, giữ theo TTL (mặc định 300s) để hỗ trợ reconnect muộn.

**Constraints**: Tương thích ngược hợp đồng SSE/API hiện tại; không thêm `RoleSlug`; không phá vỡ module boundary (chỉ qua `presentation/contract` + `application/dto` + `domain/constants`); mọi migration qua Goose.

**Scale/Scope**: Hiện một instance mỗi vai trò (http/worker). Kiến trúc transport đặt sau lớp trừu tượng để hỗ trợ mở rộng ngang sau này.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Nguyên tắc | Đánh giá | Ghi chú |
|-----------|----------|---------|
| §I Independently Deployable Services | PASS | Chỉ sửa trong `commerce` + `shared`; không đổi topology. |
| §II Strict Module Boundaries & DTO Contracts | PASS | Brand đọc dữ liệu fashion qua `IFashionContract` + DTO; thêm method thuần `IsWearableCategory` (chỉ đọc repo của fashion). Không import entity/repo chéo module. |
| §II Pure Contract Usecase | PASS | Method contract mới của fashion chỉ dùng repo/mapper của chính fashion. |
| §III Async Event-Driven + Outbox | PASS | Event `fashion.event.processing` publish qua `IEventPublisher` trong `uow.Execute` (outbox). SSE last-mile dùng Redis Pub/Sub (không phải write side-effect → không thuộc outbox). |
| §IV Clean Architecture & Layer Dependency | PASS | Handler mỏng → usecase; usecase dùng repo/contract; không skip layer. |
| §V Transaction & Error Handling | PASS | Dùng `apperror` factory + `uow.Execute`; transaction cho claim/confirm. |
| §VI AI Pipeline & Cost Control | PASS | Retry thủ công chỉ từ trạng thái hợp lệ (claim nguyên tử chống trùng); chi phí AI kiểm soát bởi quota hiện hữu. |
| §VII Design Patterns Compliance | PASS | Tuân theo `docs/system-design/design-pattern.md`, messaging-guide. |
| §VIII Infrastructure Portability & No Vendor Lock-in | PASS | Redis là hạ tầng chung, đã có ở cả self-hosted và managed; không dùng tính năng vendor-exclusive. |
| Technology Stack / Goose Migrations | PASS | Thay đổi schema qua Goose migration (service `commerce`). |
| Code Style / Language / RBAC | PASS | Không thêm RoleSlug; message tiếng Việt; DTO camelCase. |

**Kết luận gate**: Không có vi phạm cần justify → Complexity Tracking để trống.

## Project Structure

### Documentation (this feature)

```text
specs/023-analyze-status-handling/
├── spec.md              # /speckit-specify output
├── plan.md              # This file (/speckit-plan output)
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── frontend-guide.md    # Tài liệu bàn giao cho frontend (FR-017)
├── checklists/
│   └── requirements.md  # Spec quality checklist
├── contracts/
│   ├── api-contracts.md
│   ├── event-contracts.md
│   └── realtime-transport.contract.md
└── tasks.md             # Phase 2 output (/speckit-tasks — không tạo ở đây)
```

### Source Code (repository root)

```text
shared/
├── application/
│   ├── constants/
│   │   ├── analyzetaskstatus/analyzetaskstatus.go      # + Processing
│   │   └── eventconstants/{eventconstants.go,queues.go} # + processing topic/queue
│   └── sse/
│       ├── sse_manager.go                               # + replay buffer
│       └── realtime.go                                  # [NEW] IRealtimePublisher/IPublisher
├── infrastructure/
│   ├── messaging/
│   │   ├── redis_realtime_publisher.go                  # [NEW] Redis Pub/Sub impl
│   │   ├── rabbitmq_constants.go                        # + routing keys
│   │   └── topology.go                                  # + bindings + DLQ
│   └── caching/redis.go                                 # reuse *redis.Client
├── presentation/
│   └── sse_relay.go                                     # [NEW] Redis→local relay lifecycle
└── config/{config.go,loader.go,validation.go}           # + SSE replay buffer config

services/commerce/
├── database/migrations/
│   └── <ts>_add_brand_item_failed_status.sql            # [NEW] (brand status CHECK)
└── internal/
    ├── bootstrap/app.go                                 # start/stop SSE relay (http|all)
    ├── di/{wire.go,wire_gen.go}                         # wire relay + publisher
    └── modules/
        ├── wardrobe/
        │   ├── domain/
        │   │   ├── repositories/interfaces.go           # + ConfirmReview/claim
        │   │   └── constants/wardrobestatus/…           # (no change)
        │   ├── application/
        │   │   ├── errors/errors.go                     # + confirm/retry errors
        │   │   ├── interface/usecase/wardrobe_uc.go     # + ConfirmWardrobeItemReview
        │   │   └── usecase/wardrobe/item/item_uc_write.go
        │   │   └── usecase/wardrobe/fashion_event_uc.go # + HandleFashionProcessing
        │   ├── infrastructure/persistence/wardrobe_repo.go
        │   └── presentation/{handler,router}            # + confirm-review route
        ├── brand/
        │   ├── domain/constants/branditem/branditemstatus # + Failed
        │   ├── application/
        │   │   ├── dto/brand_item.go                    # + reason fields
        │   │   ├── usecase/item/{crud_uc.go,fashion_event_uc.go}
        │   │   └── interface/usecase/…                  # + retry UC
        │   ├── infrastructure/persistence/…             # + claim retry
        │   └── presentation/{handler,router}            # + retry route
        └── fashion/
            ├── application/dto/{fashion.go,contract_dto.go,event_payload.go}
            ├── application/mapper/…                     # + reasons mapping
            ├── application/usecase/worker/analyze_uc.go # + processing event, PreserveCategory
            └── presentation/contract/fashion_contract*.go # + IsWearableCategory
```

**Structure Decision**: Giữ nguyên layout multi-module hiện có; thay đổi tập trung ở service `commerce` + thư viện `shared`. Không tạo service/module mới.

## Complexity Tracking

> Không có vi phạm Constitution Check cần justify.

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| — | — | — |

## Phases

- **Phase 0 — Outline & Research**: [research.md](./research.md) — đã giải quyết toàn bộ unknown (transport, migrations, brand status, contract surface).
- **Phase 1 — Design & Contracts**:
  - [data-model.md](./data-model.md)
  - [contracts/api-contracts.md](./contracts/api-contracts.md)
  - [contracts/event-contracts.md](./contracts/event-contracts.md)
  - [contracts/realtime-transport.contract.md](./contracts/realtime-transport.contract.md)
  - [quickstart.md](./quickstart.md)
- **Phase 2 — Tasks**: tạo bởi `/speckit-tasks` (không thuộc lệnh này).

## Post-Design Constitution Re-check

Sau thiết kế: không phát sinh vi phạm mới. Các contract mới giữ đúng ranh giới module (brand → `IFashionContract`), event mới đi qua outbox, migration dùng Goose, không thêm RoleSlug. **PASS**.
