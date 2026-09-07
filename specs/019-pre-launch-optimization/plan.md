# Implementation Plan: Tối Ưu Hóa Pre-Launch — Hardening, Hiệu Năng và Độ Bảo Trì

**Branch**: `019-pre-launch-optimization` | **Date**: 2026-09-06 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/019-pre-launch-optimization/spec.md`

## Summary

Gói tối ưu pre-launch gồm 5 hạng mục: (1) harden HTTP server (timeouts + body-limit 3 tầng 2/10 MB + registry route free), (2) bộ test bảo đảm luồng tiền và phân quyền (unit + integration), (3) cache tổng quan gói cước cho AI hot path, (4) nén phản hồi + xác minh migration lock đa instance, (5) hygiene pass (tách 2 file ~500 dòng, dọn fmt.Println). Toàn bộ giữ zero breaking change JSON API.

## Technical Context

**Language/Version**: Go 1.25+ (constitution), Google Wire DI

**Primary Dependencies**: Gin, GORM, goose (migration, đã dùng session locker), go-redis, RabbitMQ client tự viết, Elasticsearch client, zap logger. Bổ sung mới: `github.com/gin-contrib/gzip`

**Storage**: PostgreSQL (schema-per-module), Redis (cache), Elasticsearch (search). **Không có migration schema mới trong feature này** — chỉ đọc/verify cơ chế hiện hành.

**Testing**: `go test -race` (unit, Makefile `test`/`test-race`) + integration test theo pattern env-skip `TEST_DATABASE_URL` (Makefile `test-integration` — sẽ mở rộng phạm vi chạy thêm module subscription)

**Target Platform**: Linux server (deploy) / Windows dev (Git Bash); HTTP public qua gateway

**Performance Goals**: AI hot path giảm từ 2 → ≤1 lượt truy vấn tổng quan gói/request; response JSON dashboard giảm ≥50% kích thước truyền khi nén; timeout/body-limit không tăng p95 của request hợp lệ

**Constraints**: Zero breaking change JSON (FR-013/SC-008); không phá vỡ SSE streaming (FR-003); cache không chứa số đếm hạn mức (FR-007); tuân thủ Hiến pháp §II/§III/§IV/§V/§VI

**Scale/Scope**: 5 user story, ~10 file production mới/sửa + ~15 file test + cấu hình; không đụng schema DB

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Nguyên tắc | Áp dụng trong feature này | Trạng thái |
|---|---|---|
| §I Pragmatic Modular Monolith | Không tạo module mới; middleware dùng chung đặt tại `internal/shared/presentation/middleware` | ✅ |
| §II Strict Module Boundaries & DTO Contracts | Cache tổng quan gói là cơ chế **intra-module** của subscription (interface ở `application/interface/caching`, impl ở `infrastructure/caching`); không import chéo ngoài danh mục cho phép; caller AI vẫn đi qua `IUserSubscriptionContract` như cũ | ✅ |
| §III Event-Driven + Outbox | Invalidation cache ưu tiên gọi trực tiếp tại các điểm mutation trong cùng module (purchase apply, renewal, admin set plan, auto-renew); không tạo kênh message mới; `TopicSubscriptionPlanChanged` hiện có giữ nguyên vai trò cho aiexecutor | ✅ |
| §IV Clean Architecture & Layer Dependency | Body-limit/gzip middleware ở tầng shared presentation; cache interface ở application, impl ở infrastructure; không layer skipping | ✅ |
| §V Transaction & Error Handling | 413 body-limit trả về qua `apperror` factory mới (413) theo đúng pattern constructor hiện hành; timeout không đụng UoW | ✅ |
| §VI AI Pipeline & Cost Control | Không đụng reservation-first/quota logic; cache chỉ phục vụ tổng quan gói (đọc), quota vẫn ghi trực tiếp từng request | ✅ |
| §VII design-pattern.md compliance | Middleware mới phải tuân Handler Pattern (error-returning + WrapHandler nếu là handler); test tuân theo pattern hiện hành | ✅ |

**Post-design re-check**: xem cuối research.md §Constitution Re-check — không có vi phạm mới; body-limit dùng `http.MaxBytesReader` ở tầng middleware nên handler/usecase không đổi chữ ký (không phá Handler Pattern).

## Project Structure

### Documentation (this feature)

```text
specs/019-pre-launch-optimization/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output
│   └── http-behavior.md # Hợp đồng hành vi HTTP (timeouts, body-limit, gzip, 413)
└── tasks.md             # Phase 2 output (/speckit-tasks - NOT created here)
```

### Source Code (repository root)

```text
internal/
├── config/
│   └── config.go                          # +Server: ReadTimeout/WriteTimeout/IdleTimeoutSeconds,
│                                          #  +BodyLimit: DefaultMB/ElevatedMB/FreeRoutes[] (+ config yaml mẫu)
├── api/routes/router.go                   # đăng ký BodyLimitMiddleware + gzip (loại trừ route stream)
├── bootstrap/app.go                       # http.Server gắn timeouts; thay fmt.Println bằng logger
├── shared/
│   ├── presentation/middleware/
│   │   ├── body_limit.go                  # MỚI: 3-tier body limit (MaxBytesReader + tier table + free registry)
│   │   └── body_limit_test.go             # unit: 413, tier phân route, free route bỏ qua
│   ├── infrastructure/db/
│   │   └── migration.go                   # thay fmt.Println bằng logger; giữ session locker
│   └── infrastructure/caching/            # (không đổi, tham chiếu)
├── modules/subscription/
│   ├── application/interface/caching/
│   │   └── overview_cache.go              # MỚI: ISubscriptionOverviewCache (Get/Delete theo userID)
│   ├── application/usecase/subscription/  # usecase đọc: check cache trước; mutation: invalidate
│   ├── application/mapper/subscription.go # map ContractDTO <-> cache entry
│   ├── infrastructure/caching/
│   │   └── redis_overview_cache.go        # MỚI: redis impl TTL cấu hình (mặc định 60s)
│   ├── presentation/contract/subscription_contract_impl.go # đọc qua cache, không đổi chữ ký
│   ├── application/usecase/purchase/      # unit test mới (mock) + invalidate tại apply
│   ├── application/usecase/payment/       # integration test webhook reconciliation (TEST_DATABASE_URL)
│   └── application/usecase/subscription/  # integration test renewal workflow + unit tests
├── modules/identity/presentation/middleware/  # test phân quyền router (RBAC matrix)
├── modules/wardrobe/presentation/handler/     # tách wardrobe_item_handler.go (501 dòng) theo nhóm route
└── modules/subscription/application/usecase/subscription/
    └── subscription_renewal_workflow.go       # tách (495 dòng) + extract executePaidRenewal (90 dòng)

Makefile                                   # mở rộng test-integration chạy thêm ./internal/modules/subscription/...
go.mod                                     # + github.com/gin-contrib/gzip
```

**Structure Decision**: Giữ nguyên cấu trúc module hiện hành (constitution §I/§IV) — mọi file mới nằm đúng vị trí layer của module/tầng shared tương ứng; không tạo package ngoài quy hoạch.

## Complexity Tracking

> Không có vi phạm constitution cần justification — bảng trống.

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| — | — | — |
