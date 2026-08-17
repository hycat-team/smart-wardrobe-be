# Implementation Plan: Brand RBAC Tập Trung & Bổ Sung Nghiệp Vụ Quản Trị Admin

**Branch**: `012-brand-rbac-admin` | **Date**: 2026-08-17 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification từ `specs/012-brand-rbac-admin/spec.md`

## Summary

**Yêu cầu chính**: (1) Tập trung Brand RBAC (owner/staff) về middleware thay vì check rải rác trong ~30 usecase, kèm cache membership để giảm truy vấn DB; (2) Đóng lỗ hổng IDOR brand dashboard; (3) Bổ sung 6 nhóm nghiệp vụ quản trị admin (payment resolution, subscription/plan, user admin, brand member admin, AI executor admin, audit log); (4) Tài liệu hóa ma trận quyền owner/staff; (5) Đồng bộ mâu thuẫn docs-code theo chuẩn code và ghi out-of-scope vào docs.

**Hướng kỹ thuật**:
- Middleware brand mới (`BrandRoleMiddleware`) đặt trong `brand/presentation/middleware/`, expose qua contract `IBrandRoleMiddleware` cho dashboard dùng (tuân thủ module boundary). Hai variant: lấy `brandId` từ path (portal) và từ query (dashboard); option `AllowAdmin` (portal=false, dashboard=true). Lưu `brandId`/`brandRole` vào context theo chuẩn `contextutils`.
- Cache membership qua Redis theo `userID` (map `brandID→role` cho membership active của brand active), read-through trong `BrandUseCase.GetUserBrandRole`, TTL mặc định 5 phút + invalidation tại các điểm thay đổi membership/brand; fallback DB khi Redis lỗi.
- Bổ sung handler/router admin ở từng module sở hữu nghiệp vụ (subscription, identity, brand, aiexecutor); dựng tầng HTTP mới cho module aiexecutor; tạo module mới `admin` cho audit log (entity/repo/service/contract/consumer worker/cleanup worker) — **ghi audit bất đồng bộ qua outbox + RabbitMQ** (handler ghi outbox record → OutboxWorker publish → consumer module `admin` ghi `admin_audit_logs`).
- 1 migration Goose duy nhất gộp: thêm cột theo dõi phiên vào `identity.refresh_tokens` + tạo bảng `shared.admin_audit_logs`.
- Docs: `docs/system-design/brand-permission-matrix.md` (ma trận quyền), sửa 4 file docs mâu thuẫn, ADR out-of-scope.

## Technical Context

**Language/Version**: Go (golang v1.25+)

**Primary Dependencies**: gin-gonic (HTTP), gorm.io/gorm (ORM), google/wire (DI), redis/go-redis/v9 (cache), swaggo/swag (swagger), pressly/goose (migrations), rabbitmq (amqp) cho worker/queue (đã có sẵn).

**Storage**: PostgreSQL (multi-schema: identity, wardrobe, fashion, brand, subscription, aiexecutor, dashboard, shared) + Redis (cache membership, OTP, pricing).

**Testing**: `go test ./...` (make test); toàn bộ cổng chất lượng `make check` (fmt, tidy, generate-check, vet, test, build).

**Target Platform**: Linux server (single binary, Docker multi-stage).

**Project Type**: modular monolith web service (REST API backend).

**Performance Goals**: ≥90% request portal/dashboard không truy vấn DB để kiểm tra quyền (cache hit) khi cache hoạt động bình thường (SC-003).

**Constraints**: Tuân thủ hiến pháp (module boundary qua contract + DTO, layer inward, DI qua Wire, migration Goose, apperror + message tiếng Việt, DTO JSON camelCase, outbox cho write side-effect xuyên module). Không RBAC toàn hệ thống (giữ `users.role_slug` admin/user).

**Scale/Scope**: 8 module; ~130 endpoint hiện có; ~30 endpoint admin mới; 1 migration gộp; 1 module mới (`admin`); 1 module bổ sung tầng HTTP (aiexecutor).

## Constitution Check

*GATE — phải pass trước Phase 0; re-check sau Phase 1.*

| Gate | Yêu cầu hiến pháp | Thiết kế đáp ứng |
|------|-------------------|------------------|
| G1 | Cross-module chỉ qua `presentation/contract` + DTO; không import usecase/entity/repo/persistence module khác | Dashboard dùng `IBrandRoleMiddleware` (brand/presentation/contract) — không import file middleware trực tiếp; các module ghi audit dùng `IAuditLogContract` (admin/presentation/contract); identity tạo user dùng usecase nội bộ; brand dùng `identityContract` (đã có) |
| G2 | Layer inward `presentation → application → domain ← infrastructure` | Middleware (presentation) phụ thuộc `IBrandUseCase` (application interface); cache service ở `brand/infrastructure/caching` implement interface ở `brand/application/interface/caching`; handler admin gọi usecase interface, không gọi infra |
| G3 | Mọi thay đổi DB qua Goose migration | 1 migration gộp: `..._add_session_tracking_and_admin_audit_logs.sql` |
| G4 | Write side-effect xuyên module ưu tiên event/outbox; grant AI đồng bộ qua cơ chế hiện có | Cấp/hủy gói admin gọi luồng `PlanChanged` hiện có (queue worker); **ghi audit qua outbox + RabbitMQ** (handler ghi outbox → OutboxWorker publish → consumer ghi bảng) — không gọi thẳng cross-module |
| G5 | apperror factory + message client tiếng Việt | Mọi lỗi mới dùng `apperror.NewForbidden/NewBadRequest/NewNotFound...`; swagger/message tiếng Việt |
| G6 | DTO JSON camelCase cho frontend | Request/response DTO mới dùng json tag camelCase |
| G7 | Không FK xuyên module ngoài `users` | `admin_audit_logs.actor_user_id` FK→identity.users (cho phép); không FK xuyên module khác |
| G8 | Mapper trong `application/mapper`; single-use helper `_helper.go` | Tuân thủ khi thêm mapper/helper mới |

**Kết quả gate**: PASS — không có vi phạm cần justify (không cần Complexity Tracking).

## Project Structure

### Documentation (this feature)

```text
specs/012-brand-rbac-admin/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/
│   ├── admin-api.md              # Danh sách endpoint admin mới (theo module)
│   ├── brand-role-middleware.md  # Contract middleware brand RBAC
│   ├── admin-audit-log.md        # Contract audit log + schema payload
│   └── brand-permission-matrix.md# Ma trận quyền owner/staff (nguồn cho docs)
└── tasks.md             # Phase 2 output (/speckit-tasks) — chưa tạo
```

### Source Code (repository root)

```text
internal/
├── modules/
│   ├── brand/
│   │   ├── application/
│   │   │   ├── interface/caching/brand_membership_cache.go     # MỚI: interface cache
│   │   │   ├── interface/usecase/brand_uc.go                   # SỬA: thêm invalidation methods
│   │   │   └── usecase/brand/usecase.go                        # SỬA: GetUserBrandRole read-through cache + invalidation
│   │   ├── infrastructure/caching/redis_brand_membership_cache.go # MỚI
│   │   ├── presentation/
│   │   │   ├── middleware/brand_role.go                        # MỚI: BrandRoleMiddleware
│   │   │   ├── contract/brand_role_middleware.go               # MỚI: IBrandRoleMiddleware
│   │   │   ├── contract/brand_cache.go                         # MỚI: IBrandCacheInvalidator
│   │   │   ├── handler/admin_brand_handler.go                  # MỚI: member admin
│   │   │   └── router/router.go                                # SỬA: 2 group quyền + admin member group
│   │   └── provider.go                                         # SỬA: wire middleware + cache + admin handler
│   ├── dashboard/
│   │   ├── presentation/
│   │   │   ├── router/router.go                                # SỬA: gắn QueryAuthorize brand group
│   │   │   └── provider.go                                     # SỬA: nhận IBrandRoleMiddleware
│   │   └── application/usecase/…                               # SỬA: GetPointsLiability đọc role từ context; bỏ check trùng
│   ├── subscription/
│   │   ├── domain/repositories/deposit_transaction_repo.go     # SỬA: + ListAdmin
│   │   ├── domain/repositories/user_subscription_repo.go       # SỬA: + ListAdmin filter
│   │   ├── application/interface/usecase/plan_usecase.go       # SỬA: + CRUD plan
│   │   ├── application/interface/usecase/subscription_usecase.go# SỬA: + admin list/cancel/grant
│   │   ├── application/usecase/plan/…                          # SỬA: + Create/Update/SetActive
│   │   ├── application/usecase/subscription/…                  # SỬA: + AdminList/AdminCancel/AdminGrant
│   │   ├── presentation/handler/admin_billing_handler.go       # MỚI
│   │   ├── presentation/handler/admin_subscription_handler.go  # MỚI
│   │   └── presentation/router/router.go                       # SỬA: + /admin/subscriptions group
│   ├── identity/
│   │   ├── application/interface/usecase/user_uc.go            # SỬA: + admin methods
│   │   ├── application/usecase/user_uc.go                      # SỬA: + detail/create/role/sessions
│   │   ├── application/usecase/session_uc.go                   # SỬA: ghi device/IP/UA khi login/refresh
│   │   ├── presentation/handler/admin_handler.go               # SỬA: + endpoints
│   │   └── presentation/router/router.go                       # SỬA: + admin routes
│   ├── aiexecutor/
│   │   ├── application/interface/usecase/admin_uc.go           # MỚI: AIAdminUseCase interface
│   │   ├── application/usecase/admin/admin_uc.go               # MỚI
│   │   ├── domain/repositories/ai_cost_repo.go                 # SỬA: + GetUsageByUser, ListGrantsByUser
│   │   ├── domain/repositories/pricing_repo.go                 # SỬA: + ListAdmin
│   │   ├── presentation/handler/admin_handler.go               # MỚI
│   │   ├── presentation/router/router.go                       # MỚI
│   │   └── provider.go                                         # SỬA: + handler/router/usecase
│   ├── admin/                                                  # MỚI: module audit log
│   │   ├── domain/entities/admin_audit_log.go
│   │   ├── domain/repositories/admin_audit_log_repo.go
│   │   ├── infrastructure/persistence/admin_audit_log_repo.go
│   │   ├── application/usecase/audit_log_uc.go
│   │   ├── application/interface/usecase/audit_log_uc.go
│   │   ├── presentation/contract/audit_log_contract.go        # Record = publish outbox (async)
│   │   ├── presentation/handler/admin_audit_handler.go
│   │   ├── presentation/router/router.go
│   │   ├── presentation/worker/audit_log_consumer.go           # MỚI: consume queue → ghi bảng
│   │   ├── presentation/worker/audit_cleanup_worker.go
│   │   └── provider.go
│   ├── (shared) infrastructure/messaging/                      # SỬA: + QueueAuditLog + RoutingKeyAuditLog
│   ├── (shared) application/constants/eventconstants/           # SỬA: + TopicAuditLog
├── pkg/utils/contextutils/
│   ├── key.go                                                  # SỬA: + CtxBrandId, CtxBrandRole
│   └── user.go                                                 # SỬA: + GetBrandID, GetBrandRole
├── shared/
│   └── application/constants/…                                  # SỬA: + action constants audit (nếu cần)
├── api/routes/
│   ├── provider.go                                             # SỬA: + AIExecutorRouter, AdminRouter
│   └── router.go                                               # SỬA: + Init admin/aiexecutor router
└── di/
    ├── wire.go                                                 # SỬA: + admin.ProviderSet
    └── wire_gen.go                                             # SINH LẠI qua make wire

database/migrations/
└── 20260817090000_add_session_tracking_and_admin_audit_logs.sql   # MỚI (gộp 2 thay đổi DB)

docs/
├── system-design/brand-permission-matrix.md                    # MỚI
├── decisions/out-of-scope-2026-08.md                           # MỚI
├── api/dashboard-api-v2.md                                     # SỬA
├── api/brand-api.md                                            # SỬA
├── api/constants/brand.md                                      # SỬA (reference ma trận)
├── api/loyalty-api.md                                          # SỬA
├── system-design/architecture.md                               # SỬA
├── product/roadmap.md                                          # SỬA
└── README.md                                                   # SỬA
```

**Structure Decision**: Modular monolith hiện có — giữ nguyên kiến trúc, chỉ thêm module `admin` (audit log) + tầng HTTP mới cho aiexecutor + các handler admin trong module sở hữu. Không refactor route admin hiện có (ngoài phạm vi).

## Complexity Tracking

> Không cần — Constitution Check pass không có vi phạm phải justify.