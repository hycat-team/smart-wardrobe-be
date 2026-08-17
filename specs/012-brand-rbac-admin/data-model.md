# Data Model: Brand RBAC Tập Trung & Bổ Sung Nghiệp Vụ Quản Trị Admin

**Date**: 2026-08-17 | **Feature**: [spec.md](./spec.md)

Mô hình dữ liệu cho feature: **1 migration Goose gộp** (2 thay đổi DB trong cùng 1 file) + 1 cache key Redis. Các entity hiện có được tái sử dụng không đổi.

---

## Migration duy nhất — Theo dõi phiên + bảng audit log

**File**: `database/migrations/20260817090000_add_session_tracking_and_admin_audit_logs.sql`

Gộp 2 thay đổi DB vào 1 file migration để hạn chế file nhỏ lẻ (đầy đủ `-- +goose Up` / `-- +goose Down`).

### 1. Bảng sửa: `identity.refresh_tokens` (thêm cột)

| Cột | Kiểu | Ràng buộc | Ghi chú |
|-----|------|-----------|---------|
| `device_name` | varchar(255) | NULL | Tên thiết bị (từ User-Agent parse hoặc dạng chuẩn hóa) |
| `device_ip` | varchar(45) | NULL | Địa chỉ IP (hỗ trợ IPv6) |
| `user_agent` | varchar(512) | NULL | User-Agent thô từ request |
| `last_active_at` | timestamptz | NULL | Cập nhật mỗi lần refresh token |

**Ghi nhận tại**: `Login` (tạo phiên) và `RefreshToken` (rotation — cập nhật `last_active_at`, `device_ip`, `user_agent`).

**Hành vi**: cột cũ có dữ liệu phiên trước migration → NULL cho 4 cột mới (không backfill bắt buộc; admin xem phiên cũ sẽ không có thông tin thiết bị).

---

### 2. Bảng mới: `shared.admin_audit_logs`

(Cùng file migration — nằm sau phần ALTER TABLE `identity.refresh_tokens` trong `-- +goose Up`; đảo ngược trong `-- +goose Down`.)

| Cột | Kiểu | Ràng buộc | Ghi chú |
|-----|------|-----------|---------|
| `id` | uuid | PK default `gen_random_uuid()` | |
| `actor_user_id` | uuid | NOT NULL, FK → `identity.users(id)` | Admin thực hiện hành động (cho phép FK xuyên schema tới users — ngoại lệ chuẩn dự án) |
| `action` | varchar(100) | NOT NULL | Tên hành động chuẩn hóa (snake_case), ví dụ `user.deactivate`, `brand.approve`, `payment.resolve_paid` |
| `target_type` | varchar(50) | NOT NULL | Loại đối tượng chịu tác động: `user`, `brand`, `brand_member`, `deposit_transaction`, `subscription_plan`, `user_subscription`, `ai_model_config`, `ai_policy_grant` |
| `target_id` | uuid | NULL | ID đối tượng (NULL khi không áp dụng) |
| `payload` | jsonb | NOT NULL default `'{}'` | Chi tiết: trạng thái cũ→mới, lý do, dữ liệu bổ sung (không chứa mật khẩu/token/secret) |
| `request_ip` | varchar(45) | NULL | IP của admin |
| `created_at` | timestamptz | NOT NULL default `now()` | |

**Index**:
- `idx_admin_audit_logs_actor` trên `(actor_user_id)`
- `idx_admin_audit_logs_target` trên `(target_type, target_id)`
- `idx_admin_audit_logs_created_at` trên `(created_at)` — phục vụ lọc theo thời gian + cleanup

**Retention**: cleanup worker xóa bản ghi `created_at < now() - interval '12 months'` định kỳ (hàng ngày).

**Ghi bất đồng bộ (outbox + RabbitMQ)**: bảng `admin_audit_logs` KHÔNG được ghi trực tiếp từ handler. Luồng: handler → `IAuditLogContract.Record` (ghi `outbox_records` topic `admin.audit.log`) → `OutboxWorker` publish → consumer module `admin` ghi bảng. `outbox_records` (schema `shared`) được tái sử dụng, không cần bảng outbox mới.

**Các action ghi audit** (FR-016 + clarify):
- user: `user.deactivate`, `user.activate`, `user.role_change`, `user.create`, `user.sessions_revoke`
- brand: `brand.approve`, `brand.suspend`, `brand.archive`, `brand.create_admin`
- brand_member: `brand.member_add`, `brand.member_role_change`, `brand.member_remove`, `brand.ownership_transfer`
- payment: `payment.retry_investigation`, `payment.resolve_paid`, `payment.resolve_cancelled`, `payment.resolve_creation_failed`
- plan/subscription: `subscription.plan_create`, `subscription.plan_update`, `subscription.plan_deactivate`, `subscription.cancel`, `subscription.grant`
- ai: `ai.pricing_update`, `ai.grant_create`

---

## Cache Redis — Membership brand theo user (không đổi DB)

**Key**: `brand:members:{userID}`

**Value**: JSON object map brandID→role

```json
{
  "2f1e8a8b-...-brandA": "owner",
  "5c9d3f11-...-brandB": "staff"
}
```

- Chỉ chứa membership **active** của brand **active** (member.status=ACTIVE, brand.status=ACTIVE).
- **TTL**: mặc định 5 phút (cấu hình qua config `BrandMembershipCacheTTL`).
- **Fallback**: Redis lỗi → đọc DB (`GetActiveByUserID` + filter brand active).
- **Invalidation**:
  - `InvalidateByUserID(userID)` — xóa key user (thêm member, đổi role, gỡ member, transfer ownership).
  - `InvalidateByBrandID(brandID)` — fan-out: `GetByBrandID(brandID)` → xóa key từng member (đổi trạng thái brand).
  - Gọi tại: `BrandUseCase.AddBrandMembers`, `UpdateBrandStatus`, các endpoint admin member mới.

---

## Entity mới (module `admin`)

### AdminAuditLog
| Field | Type | Ghi chú |
|-------|------|---------|
| ID | uuid | PK |
| ActorUserID | uuid | FK users |
| Action | string | action chuẩn hóa |
| TargetType | string | user/brand/... |
| TargetID | *uuid | nullable |
| Payload | map[string]any | jsonb |
| RequestIP | string | nullable |
| CreatedAt | time.Time | |

File: `internal/modules/admin/domain/entities/admin_audit_log.go`

---

## Entity sửa đổi (identity)

### RefreshToken (thêm field)
| Field | Type | Ghi chú |
|-------|------|---------|
| DeviceName | *string | mới |
| DeviceIP | *string | mới |
| UserAgent | *string | mới |
| LastActiveAt | *time.Time | mới |

File: `internal/modules/identity/domain/entities/user.go` (phần RefreshToken)

---

## Không thay đổi schema

- `subscription_plans`, `user_subscriptions`, `deposit_transactions`, `brand_members`, `ai_model_configs`, `ai_policy_grants`, `ai_usage_events` — giữ nguyên; chỉ thêm method repo đọc/lọc.
- Không thêm FK xuyên module ngoài `admin_audit_logs.actor_user_id → identity.users`.

## Quy tắc chung

- Migration viết đầy đủ `-- +goose Up` / `-- +goose Down`.
- Không sửa file init seed/init-db; chỉ thêm migration mới.
- JSON DTO camelCase; payload audit lưu jsonb dùng snake_case nội bộ (không phải DTO frontend).