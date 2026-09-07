# Data Model: Tối Ưu Hóa Pre-Launch (019)

**Date**: 2026-09-06 | **Spec**: [spec.md](spec.md) | **Research**: [research.md](research.md)

Feature này **không thay đổi schema CSDL** (không migration mới). Các "entity" dưới đây là cấu hình, entry cache trong Redis, và ma trận test.

## 1. Server Hardening Config (mới — cấu hình)

Nguồn chân lý: `internal/config/config.go` (struct `Server` mở rộng + struct `BodyLimit` mới) + file config yaml tương ứng.

| Trường | Kiểu | Mặc định | Quy tắc validation |
|---|---|---|---|
| `server.read_timeout_seconds` | int | 30 | > 0 bắt buộc ở production (validate startup) |
| `server.write_timeout_seconds` | int | 60 | > 0 ở production; đủ lớn cho stream ngắn (SSE tự gia hạn deadline) |
| `server.idle_timeout_seconds` | int | 120 | > 0 ở production |
| `server.gzip_enabled` | bool | true (production) | — |
| `server.body_limit.default_mb` | int | 2 | > 0 ở production |
| `server.body_limit.elevated_mb` | int | 10 | ≥ default_mb |
| `server.body_limit.free_routes` | []string | [] (trống) | pattern `METHOD prefix` — thêm qua config, không đụng code |

Quan hệ: được `bootstrap` tiêu thụ khi dựng `http.Server` và đăng ký middleware; không entity DB nào tham chiếu.

## 2. Body Limit Tier Table (mới — cấu trúc runtime trong middleware)

Bảng phân tầng (method + path pattern → limit), đánh giá theo thứ tự: **free registry → elevated → default**.

| Tầng | Giá trị | Patterns (từ research.md D2) |
|---|---|---|
| free | vô hạn (bỏ qua wrap) | `[]` — trống, thêm qua config |
| elevated | `elevated_mb` (10) | `POST /api/v1/ai/` · `POST /api/v1/brands/*/conversation*` · `POST|PUT|PATCH /api/v1/brand-portal/` · `POST|PUT|PATCH /api/v1/admin/brands/` · `POST /brands/claim` · `POST /brands/*/join-loyalty` · `POST /brand-items/*/feedbacks` · `POST /brand-benefits/*/redeem` |
| default | `default_mb` (2) | mọi route còn lại |

Trạng thái/lifecycle: stateless — phân tích per-request, không cache pattern.

## 3. Subscription Overview Cache Entry (mới — Redis)

Key: `sub:overview:{userID}` | TTL: `cache.overview_ttl_seconds` (mặc định 60)

| Trường | Kiểu | Ghi chú |
|---|---|---|
| plan_id | uuid | |
| plan_name / plan_slug | string | |
| expires_at | *time | |
| is_auto_renew_enabled | bool | invalidation bắt buộc khi toggle |
| max_wardrobe_items / max_outfits | int | |
| ai_outfit_daily_quota / ai_chat_daily_quota | int | cấu hình gói, KHÔNG phải số đếm sử dụng |
| cached_at | time | quan sát độ tuổi entry |

**Lifecycle**: `miss → đọc usecase → fill cache` ; `mutation (purchase apply / renewal apply / admin set plan active / set auto-renew / webhook apply) → Delete(userID)` ; lỗi Redis → bỏ qua cache (fallback direct), không fail request. **Số đếm hạn mức ngày (UserSubscriptionDTO/UserDailyQuota) tuyệt đối không nằm trong entry này** (FR-007).

## 4. Migration Lock (hiện hành — chỉ tài liệu hóa + test)

- Cơ chế: goose `WithSessionLocker(lock.NewPostgresSessionLocker())` — Postgres session-level advisory lock tại [migration.go](../../../internal/shared/infrastructure/db/migration.go).
- Hành vi mong đợi khi 2 instance chạy đồng thời: instance giữ lock chạy Up; instance kia chờ đến lượt rồi chạy Up (idempotent, goose bỏ qua migration đã áp) — hoặc thoát an toàn; không xung đột schema.
- Tự nhả: session DB chết → lock nhả tự động (tính chất advisory lock cấp session).

## 5. Test Coverage Matrix (FR-004..FR-006 ↔ test file)

| Luồng (FR) | Cấp độ | File đích (tạo mới) | Kịch bản |
|---|---|---|---|
| Purchase direct (FR-004) | unit | `subscription/application/usecase/purchase/checkout_test.go` | happy; đang có giao dịch pending → tái sử dụng link; validate thất bại |
| Renewal workflow (FR-004) | unit + **integration** | `subscription/application/usecase/subscription/renewal_test.go` + `renewal_integration_test.go` | unit: phân nhánh finite/paid; integration: apply thành công; thất bại → fallback default plan |
| Webhook reconciliation (FR-004) | **integration** | `subscription/application/usecase/payment/webhook_integration_test.go` | success apply idempotent (tái phát cùng order); trùng lặp bỏ qua an toàn; lệch số tiền |
| Quota (FR-005) | unit | `subscription/application/usecase/quota/quota_test.go` + mocks tại fashion AI test hiện hữu | vượt hạn mức bị chặn; reset theo ngày; ghi nhận sử dụng |
| RBAC (FR-006) | unit (router/middleware) | `identity/presentation/middleware/roles_authorize_test.go` + test mapping router admin/brand-portal | system admin/user × admin routes; brand owner/staff × portal routes; guest bị chặn |

Quy ước integration: file `*_integration_test.go`, skip khi thiếu `TEST_DATABASE_URL`, chạy qua `make test-integration` (Makefile mở rộng target hiện hành).

## 6. File Split Map (FR-011)

| File gốc (dòng) | Tách thành | Ranh giới |
|---|---|---|
| `wardrobe/presentation/handler/wardrobe_item_handler.go` (501) | `wardrobe_item_handler.go` + `wardrobe_catalog_handler.go` (+ stats/search nếu cần) | CRUD item ↔ system catalog/upload signature ↔ stats/search; message constants giữ theo file tương ứng |
| `subscription/application/usecase/subscription/subscription_renewal_workflow.go` (495) | `subscription_renewal_workflow.go` + `renewal_payment.go` + `renewal_fallback.go` | orchestration ↔ paid renewal (extract `executePaidRenewal` thành các bước < 80 dòng) ↔ fallback default plan |
