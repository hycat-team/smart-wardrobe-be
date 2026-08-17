# Quickstart — Hướng dẫn kiểm chứng end-to-end

**Date**: 2026-08-17 | **Feature**: [spec.md](./spec.md)

Hướng dẫn chạy và kiểm chứng feature sau khi implement. Chi tiết endpoint: [contracts/admin-api.md](./contracts/admin-api.md); dữ liệu: [data-model.md](./data-model.md).

## Prerequisites

- Go 1.25+, Docker (PostgreSQL + Redis), RabbitMQ (cho worker/queue) — xem `docs/development/local-setup.md`.
- DB đã migrate (bao gồm 1 migration mới `20260817090000_add_session_tracking_and_admin_audit_logs.sql`) và seed (`database/init/03-seed.sql` → tài khoản `admin`/`admin@smartwardrobe.com` pw `123456`).

## Setup

```powershell
# 1. Migration
make migration-up

# 2. Sinh DI + swagger
make generate

# 3. Chạy
make run
```

## Kịch bản kiểm chứng

### A. Đóng IDOR brand dashboard (US1)

1. Login user thường (vd `user@smartwardrobe.com`): `POST /api/v1/auth/login` → lấy token.
2. Tạo brand qua `POST /brand-portal/brands` (admin duyệt) hoặc dùng seed brand; ghi nhận `brandId`.
3. `GET /api/v1/brand/dashboard/kpi-cards?brandId=<brandId>` bằng token user thường (không phải member) → **kỳ vọng 403**.
4. Thêm user làm staff của brand qua `POST /brand-portal/brands/:brandId/members` (owner) → login user đó → gọi lại endpoint → **kỳ vọng 200**.
5. Login `admin` → gọi cùng endpoint → **kỳ vọng 200** (bypass).
6. `POST /api/v1/dashboard/analytics/query` với `targetDashboard=brand`, `brandId` user thường → **kỳ vọng 403**.

### B. Portal RBAC qua middleware + cache (US2)

1. User staff gọi `PUT /brand-portal/brands/:brandId/loyalty/program` → **kỳ vọng 403**.
2. User owner gọi cùng endpoint → **kỳ vọng 200**.
3. Kiểm tra Redis: `GET brand:members:{userId}` có key sau lần gọi đầu (cache hit).
4. Gỡ/đổi role member (admin hoặc owner) → gọi lại sau invalidation → quyền phản ánh ngay (không đợi TTL).

### C. Nghiệp vụ admin

**Payment resolution (3A)**
1. Tạo giao dịch ở trạng thái `investigation_required`/`reconciliation_required` (hoặc dùng data sẵn).
2. `GET /api/v1/admin/subscriptions/transactions` → có giao dịch.
3. `POST /admin/subscriptions/transactions/:orderCode/resolve-paid` với body `{reference, paymentLinkId}` → giao dịch `success`, ví/đăng ký cập nhật.
4. `resolve-paid` thiếu bằng chứng → **kỳ vọng 4xx**, ví không đổi.

**Plan & subscription (3B)**
1. `POST /api/v1/admin/subscriptions/plans` → gói mới xuất hiện trong `GET /subscriptions/plans` (public) khi active.
2. `PATCH /admin/subscriptions/plans/:id/status {isActive:false}` → gói không còn ở public list.
3. `POST /admin/subscriptions/:userId/cancel` → user về free; `POST /admin/subscriptions/:userId/grant {planId}` → user nhận gói + grant AI đồng bộ.

**User admin (3C)**
1. `POST /api/v1/admin/users` (role user) → login được.
2. `PATCH /admin/users/:id/role {role:admin}` bằng admin thường → **kỳ vọng 403**; bằng admin seed → **200**.
3. `DELETE /admin/users/:id` → user không login được, phiên bị thu hồi.
4. `GET /admin/users/:id/sessions` → có thông tin device/IP/UA (sau migration + login mới).

**Brand member admin (3D)**
1. `POST /api/v1/admin/brands/:brandId/members {emailOrUsername}` → user thành staff, vào được portal.
2. `POST /admin/brands/:brandId/transfer-ownership {newOwnerUserId, oldOwnerHandling:staff}` → user mới thành owner, chủ cũ thành staff.
3. `oldOwnerHandling:remove` → chủ cũ không vào được portal.
4. Chỉ 1 owner active → transfer/remove → **kỳ vọng 4xx**.

**AI executor admin (3E)**
1. `GET /api/v1/admin/ai/pricing` → danh sách model.
2. `PUT /admin/ai/pricing/:id {pricePerMillionTokens}` → lần tính chi phí kế tiếp dùng giá mới.
3. `GET /admin/ai/usage?userId=` → số request/token/cost; `POST /admin/ai/grants` → grant mới.

**Audit log (3F)**
1. Thực hiện 1 hành động admin (vd `PATCH /admin/users/:id/status`) → sau vài nhịp consumer (outbox → RabbitMQ → consumer ghi bảng), `GET /api/v1/admin/audit-logs` có bản ghi đúng actor/action/target. Kiểm tra thêm: `outbox_records` có record topic `admin.audit.log` rồi chuyển `published`.
2. User thường gọi `/admin/audit-logs` → **kỳ vọng 403**.
3. (Cleanup worker) bản ghi > 12 tháng bị xóa định kỳ.

### D. Ma trận quyền & docs

1. Đối chiếu ma trận tại `docs/system-design/brand-permission-matrix.md` với hành vi middleware (B) — không lệch.
2. Kiểm tra `docs/api/dashboard-api-v2.md` dùng `direct_purchase` (chữ thường), plan code `free`/`premium-monthly`.

## Kỳ vọng tổng thể

- `make check` (fmt, tidy, generate-check, vet, test, build) pass.
- Không breaking change với endpoint hiện có ngoài phạm vi.
- Cache membership hit ≥90% trên portal/dashboard khi Redis hoạt động — đo bằng counter cache hit/miss trong `RedisBrandMembershipCache` (T064); đọc qua log/metric sau một phiên chạy portal/dashboard.