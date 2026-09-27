# Implementation Plan: Xử lý kịch bản theo trạng thái phân tích trang phục AI

**Branch**: `023-analyze-status-handling` | **Date**: 2026-09-26 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/023-analyze-status-handling/spec.md`

## Summary

Bổ sung và chuẩn hóa kịch bản xử lý cho từng trạng thái phân tích AI của trang phục (B2C wardrobe + B2B brand), gồm các hạng mục (G1–G11):
1. **G1** — Endpoint xác nhận món `needs_review` → usable (`POST /wardrobe-items/{id}/confirm-review`), bắt buộc danh mục hợp lệ.
2. **G2** — Phát và hiển thị realtime trạng thái `processing` (mới) cho từng món.
3. **G3** — Thử lại an toàn: chỉ cho phép khi trạng thái hợp lệ (thất bại/cần rà soát), chống phiên trùng bằng claim nguyên tử (không cooldown/giới hạn số lần).
4. **G4** — Giữ danh mục người dùng đã sửa khi phân tích lại.
5. **G5/G6** — Brand: thêm trạng thái `failed` (+ lý do), chặn đổi khỏi `failed` bằng thao tác thủ công, thêm endpoint retry-analysis.
6. **G7** — Buffer phát lại sự kiện realtime khi client kết nối lại.
7. Tài liệu playbook + tài liệu tích hợp frontend (`frontend-guide.md`).
8. **G8** — Sửa hồi quy lọc theo danh mục: phân biệt tường minh "không lọc" với "lọc nhưng rỗng" bằng cờ `filterByFashionItems` ở tầng repository; danh mục/từ khóa không khớp ⇒ trả rỗng (short-circuit), không lọc ⇒ trả toàn bộ. Áp dụng cho cả 3 đường (US5, FR-019..FR-024).
9. **G9** — Realtime best-effort + observability: lỗi phát/relay sự kiện chỉ log cảnh báo, KHÔNG chặn nghiệp vụ (DB vẫn là nguồn sự thật); thêm counter/gauges (publish/relay thành công–thất bại, số subscriber/phiên) theo pattern metrics in-memory hiện có, phơi qua health endpoint (FR-025, FR-026).
10. **G10** — Phân loại ảnh không hợp lệ + mã hóa lý do + đồng bộ brand exhausted (US6, FR-027..FR-036): ảnh đa món/toàn thân ⇒ `failed` + mã lý do (không confirm/retry); `needs_review` chỉ còn `uncertain_category` (confirmable); tách tín hiệu AI `is_single_item`/`category_confident`; reason là mã và chỉ nằm trong `fashionItem` lồng nhau; contract DTO camelCase; brand nhận `processing-exhausted` ⇒ `failed`; quét xử lý quá hạn theo fashion item cho cả wardrobe+brand.
11. **G11** — Bỏ `confirm-review`, gộp vào `retry-analysis` kèm `categoryId` (US1 viết lại, FR-001..003/037..041): `needs_review` (`uncertain_category`) xử lý bằng retry kèm danh mục (bắt buộc, validate wearable); chế độ **danh mục cố định** (prompt metadata-only, bỏ gate `is_single_item`/`category_confident`, giữ danh mục) ⇒ hết vòng lặp + metadata/embedding đầy đủ; prompt **fail-fast** (1 call, không `ResponseSchema`); brand nhận `categoryId` qua retry request.

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

**Observability (G9)**: Realtime best-effort — lỗi publish/relay chỉ log `warn` kèm `taskId`, không làm fail nghiệp vụ. Counter/gauges in-memory (publish/relay success–failed, số subscriber theo phiên) theo đúng pattern `shared/infrastructure/messaging/metrics.go` (`atomic.Int64` + `GetMetrics()`), phơi qua payload health (`/api/v1/commerce/health`). Không thêm Prometheus/OTel mới.

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
| §V Transaction & Error Handling | PASS | Dùng `apperror` factory + `uow.Execute`; transaction cho claim/retry. |
| §VI AI Pipeline & Cost Control | PASS | Retry thủ công chỉ từ trạng thái hợp lệ (claim nguyên tử chống trùng); chi phí AI kiểm soát bởi quota hiện hữu. |
| §VII Design Patterns Compliance | PASS | Tuân theo `docs/system-design/design-pattern.md`, messaging-guide. |
| §VIII Infrastructure Portability & No Vendor Lock-in | PASS | Redis là hạ tầng chung, đã có ở cả self-hosted và managed; không dùng tính năng vendor-exclusive. |
| Technology Stack / Goose Migrations | PASS | Thay đổi schema qua Goose migration (service `commerce`). |
| Code Style / Language / RBAC | PASS | Không thêm RoleSlug; message tiếng Việt; DTO camelCase. |
| §IV + §II (G8 — lọc danh mục rỗng) | PASS | Chỉ đổi chữ ký repository nội bộ module `wardrobe` (thêm cờ `filterByFashionItems`) và short-circuit ở usecase; không thêm import chéo module, không đổi hợp đồng API, không đổi schema. |
| §III + §VII (G9 — best-effort + observability) | PASS | Lỗi phát/relay sự kiện chỉ log `warn`, không chặn write side-effect (outbox vẫn là write path). Metrics in-memory theo pattern `messaging/metrics.go`; không thêm dependency/hạ tầng mới, không đổi hợp đồng client. |
| §II + §IV + §VII (G10 — phân loại ảnh + mã lý do + brand exhausted) | PASS | Reason thuộc fashion item, hiển thị nested (không rò rỉ entity chéo module); contract DTO camelCase; brand nhận `processing-exhausted` qua RabbitMQ + outbox (không gọi chéo); quét xử lý quá hạn do fashion sở hữu + lookup owner qua contract. Không thêm RoleSlug; không migration. |
| §II + §IV (G11 — gộp confirm vào retry kèm danh mục) | PASS | Bỏ endpoint confirm; `retry-analysis` nhận `categoryId` + validate qua `IFashionContract.IsWearableCategory`; ghi danh mục qua `UpdateFashionItemCategory`; chế độ danh mục cố định dùng repo/mapper nội bộ fashion + prompt riêng; không gọi chéo entity/repo; breaking API được ghi trong tài liệu frontend. Không migration. |

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
├── playbook.md          # Playbook vận hành theo trạng thái phân tích (FR-014, SC-007)
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
│       ├── sse_manager.go                               # + replay buffer + subscriber gauge
│       └── realtime.go                                  # [NEW] IRealtimePublisher/IPublisher
│       └── realtime_metrics.go                          # [NEW] G9: counters/gauges in-memory
├── infrastructure/
│   ├── messaging/
│   │   ├── redis_realtime_publisher.go                  # [NEW] Redis Pub/Sub impl (best-effort publish)
│   │   ├── rabbitmq_constants.go                        # + routing keys
│   │   └── topology.go                                  # + bindings + DLQ
│   └── caching/redis.go                                 # reuse *redis.Client
├── presentation/
│   └── sse_relay.go                                     # [NEW] Redis→local relay lifecycle (best-effort)
└── config/{config.go,loader.go,validation.go}           # + SSE replay buffer config

services/commerce/
├── database/migrations/
│   └── <ts>_add_brand_item_failed_status.sql            # [NEW] (brand status CHECK)
└── internal/
    ├── bootstrap/app.go                                 # start/stop SSE relay (http|all)
    ├── bootstrap/router.go                              # G9: health payload + realtime metrics
    ├── di/{wire.go,wire_gen.go}                         # wire relay + publisher
    └── modules/
        ├── wardrobe/
        │   ├── domain/
        │   │   ├── repositories/interfaces.go           # + claim retry; + filterByFashionItems signature
        │   │   └── constants/wardrobestatus/…           # (no change)
        │   ├── application/
        │   │   ├── errors/errors.go                     # + retry errors (ErrRetryCategoryRequired)
        │   │   ├── interface/usecase/wardrobe_uc.go     # + RetryWardrobeAnalysis (categoryId, G11)
        │   │   ├── usecase/wardrobe/item/item_uc_write.go
        │   │   ├── usecase/wardrobe/item/item_uc_read.go    # G8: short-circuit lọc rỗng + truyền cờ
        │   │   ├── usecase/wardrobe/catalog/catalog_uc.go   # G8: truyền cờ filterByFashionItems
        │   │   └── usecase/wardrobe/fashion_event_uc.go     # + HandleFashionProcessing
        │   ├── infrastructure/persistence/wardrobe_repo.go  # G8: buildUserWardrobeFilterQuery/buildWardrobeItemSearchQuery + cờ
        │   └── presentation/{handler,router}            # + retry-analysis nhận body categoryId (G11)
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

- **Phase 0 — Outline & Research**: [research.md](./research.md) — đã giải quyết toàn bộ unknown (transport, migrations, brand status, contract surface, lọc danh mục rỗng R11, best-effort + observability R12).
- **Phase 1 — Design & Contracts**:
  - [data-model.md](./data-model.md)
  - [contracts/api-contracts.md](./contracts/api-contracts.md)
  - [contracts/event-contracts.md](./contracts/event-contracts.md)
  - [contracts/realtime-transport.contract.md](./contracts/realtime-transport.contract.md)
  - [quickstart.md](./quickstart.md)
- **Phase 2 — Tasks**: tạo bởi `/speckit-tasks` (không thuộc lệnh này).

## Post-Design Constitution Re-check

Sau thiết kế: không phát sinh vi phạm mới. Các contract mới giữ đúng ranh giới module (brand → `IFashionContract`), event mới đi qua outbox, migration dùng Goose, không thêm RoleSlug. G8 chỉ đổi chữ ký repository nội bộ module `wardrobe`; G9 dùng metrics in-memory theo pattern sẵn có và xử lý best-effort (không chặn write path). G10: reason thuộc fashion item (nested), brand nhận `processing-exhausted` qua RabbitMQ+outbox, quét stale do fashion sở hữu + lookup owner qua contract, không migration; mặc định an toàn `category_confident=false` khi thiếu và ưu tiên tính hợp lệ ảnh khi tín hiệu AI mâu thuẫn; thử lại publish lỗi ghi reason `analysis_temporary_error`. G11: bỏ endpoint confirm (breaking, ghi rõ tài liệu FE); `needs_review` xử lý bằng retry kèm danh mục; chế độ danh mục cố định dùng repo/mapper/prompt nội bộ fashion; không migration, không thêm RoleSlug. **PASS**.
