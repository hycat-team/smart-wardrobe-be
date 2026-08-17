# PROGRESS — Brand RBAC & Admin Features (spec 012)

> **Điểm tiếp tục (RESUME POINT):** bắt đầu từ **T028 (Phase 7 / US5)** — file này là nguồn trạng thái chính thức.
> Cập nhật lần cuối: **2026-08-17** (trước khi agent dừng vì rate-limit).

## Cách dùng
- Sau khi reset rate-limit: bảo agent đọc file này + `tasks.md`, tiếp tục task đầu tiên chưa `[X]` theo thứ tự `tasks.md`.
- Khi hoàn thành 1 task: đổi `[ ]` → `[X]` trong `tasks.md`, chạy build/test, rồi cập nhật mục "Trạng thái hiện tại" ở cuối file này.

## Nguyên tắc chung (bắt buộc)
- Tuân thủ AGENTS.md + `docs/development/` + `docs/system-design/`. Tiếng Việt khi giao tiếp với dev.
- Layer: `presentation → application → domain ← infrastructure`. Cross-module chỉ qua `presentation/contract/` + DTO.
- Sau khi thêm/sửa handler: chạy `make wire` + `make swagger`. Build: `go build ./...`. Format: `gofmt -l <file>`.
- Router admin cần `auth.Handle()` + `RolesAuthorize(roleslug.Admin)`, gói trong group `/admin/...`.

## Đã hoàn thành (T001–T027)
- **T001–T003**: Migration `database/migrations/20260817090000_add_session_tracking_and_admin_audit_logs.sql` (session columns refresh_tokens + `shared.admin_audit_logs`); `internal/pkg/utils/contextutils/{key.go, user.go}` thêm `CtxBrandId`, `CtxBrandRole` + getters.
- **T004–T007**: Cache membership Redis (`brand:members:{userID}`), config `brand_membership.ttl: 5m`, `GetUserBrandRole` read-through + invalidation trong `internal/modules/brand/application/usecase/brand/usecase.go`.
- **T008–T010**: `BrandRoleMiddleware` (`PathAuthorize`=no bypass, `QueryAuthorize`=admin bypass) trong `internal/modules/brand/presentation/middleware/brand_role.go` + contract; wire cache+middleware trong brand providers.
- **T011–T014 (US1)**: Dashboard router `QueryAuthorize("brandId", Owner, Staff)` 2 groups; `GetPointsLiability(ctx, userID, brandID, role)`; fix IDOR.
- **T015–T020 (US2)**: Cache invalidation `AddBrandMembers`/`UpdateBrandStatus`; brand router 3 groups (portalMember/portalOwner/admin); xóa `RequireBrandRoleShared` khỏi 5 usecase brand; chat test rewrite.
- **T021–T023 (US3)**: `docs/system-design/brand-permission-matrix.md` + references trong `docs/api/constants/brand.md`, `docs/api/brand-api.md`, `docs/api/loyalty-api.md`.
- **T024–T027 (US4)**: ĐÃ HOÀN THÀNH — xem mục "Ghi chú US4" dưới đây.

## Ghi chú US4 (đã build thành công)
- Build toàn project OK (`go build ./...` không lỗi), `make wire` đã regenerate `internal/di/wire_gen.go`.
- **Lưu ý quan trọng**: `DepositStatus` là `int16` (KHÔNG phải string), dùng helper `parseDepositStatus(name string)` trong handler để map tên → value.
- `validation` package KHÔNG có `BindUri` → dùng `c.ShouldBindUri(&params)` trực tiếp.
- File mới tạo:
  - `internal/modules/subscription/application/interface/usecase/admin_billing_usecase.go` — `AdminBillingFilter{Status *DepositStatus, FromDate, ToDate *time.Time, OrderCode *int64, UserID *uuid.UUID}` + `IAdminBillingUseCase{ListTransactions, GetTransaction, GetTransactionByOrderCode}`.
  - `internal/modules/subscription/application/usecase/admin_billing/admin_billing_uc.go` — package name là **`admin`** (không phải `admin_billing`); import trong `application/provider.go` phải alias `admin_billing "..."`.
  - `internal/modules/subscription/presentation/handler/admin_billing_handler.go` — `AdminBillingHandler{logger, adminBilling IAdminBillingUseCase, resolution IPaymentResolutionUseCase}`; `buildResolutionInput` map Evidence từ tx hiện tại (tx.Amount là `decimal.Decimal`).
- Wire: thêm `admin_billing.NewAdminBillingUseCase` vào `application/provider.go`; `handler.NewAdminBillingHandler` vào `internal/modules/subscription/provider.go` (module root).
- Router: group `/admin/subscriptions/transactions` trong `internal/modules/subscription/presentation/router/router.go`.

## Việc còn lại (đánh dấu trong tasks.md)
- **US5** T028–T032: Plan CRUD + subscription admin (`AdminSubscriptionHandler`).
- **US6** T033–T037: Session tracking (login/refresh) + user admin.
- **US7** T038–T040: Brand member admin (`AdminBrandHandler`).
- **US8** T041–T044: AI executor admin (HTTP layer + AppRouter/NewEngine/wire).
- **US9** T045–T051: Module `admin` (outbox + RabbitMQ `admin.audit.log`/`audit_log_queue`), consumer + cleanup worker, inject `IAuditLogContract` vào các admin handler.
- **US10** T052–T056: Đồng bộ docs-code + ADR out-of-scope.
- **Polish** T057–T064: `make check`, quickstart A–D, tests, cache-hit metric (SC-003 ≥90%).

## Trạng thái hiện tại
- Tiến trình: **US4 hoàn tất, build sạch**. Điểm tiếp theo: **T028** (`ISubscriptionPlanUseCase`: `CreatePlan`, `UpdatePlan`, `SetPlanActive` — chống trùng slug qua `GetBySlug`; inactive không hiện public).
