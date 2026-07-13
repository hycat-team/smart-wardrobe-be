# Phase 05a - Brand Core

## Trạng thái

Đã hoàn thành phần schema, module runtime `brand`, API core và DI cho sub-phase 05a. Chưa chạy `migrate up`.

## Files changed

- `migrations/20260628130423_brand_core.sql`
- `internal/shared/domain/entities/brand_entities.go`
- `internal/shared/domain/entities/table_name.go`
- `internal/shared/domain/constants/brandstatus/brand_status.go`
- `internal/shared/domain/constants/brandmemberrole/brand_member_role.go`
- `internal/shared/domain/constants/brandmemberstatus/brand_member_status.go`
- `internal/shared/domain/constants/brandcustomerstatus/brand_customer_status.go`
- `internal/shared/domain/constants/brandcustomerjoinedsource/brand_customer_joined_source.go`
- `internal/modules/brand/**`
- `internal/api/routes/brand/router.go`
- `internal/api/routes/provider.go`
- `internal/api/routes/router.go`
- `internal/di/wire.go`
- `internal/di/wire_gen.go`
- `api/swagger/docs.go`
- `api/swagger/swagger.json`
- `api/swagger/swagger.yaml`

## Migrations added

- `migrations/20260628130423_brand_core.sql`

Migration tạo:

- `brands`
- `brand_members`
- `brand_customers`
- partial unique index `brand_customers(brand_id, user_id)` khi `user_id IS NOT NULL`
- partial unique index `brand_customers(brand_id, phone_hash)` khi `phone_hash IS NOT NULL`
- audit fields cho `brands`: `created_by_user_id`, `approved_by_user_id`, `approved_at`

`brands.status` hiện có:

- `PENDING_REVIEW`
- `ACTIVE`
- `SUSPENDED`
- `ARCHIVED`

`brand_customers.joined_source` không dùng `STAFF_CREATED`; MVP dùng:

- `SELF_JOIN`
- `OFFLINE_PURCHASE`

## APIs added/changed

- `GET /api/v1/brands`
- `POST /api/v1/brands/:brandId/join-loyalty`
- `POST /api/v1/brand-portal/brands`
- `GET /api/v1/brand-portal/brands/:brandId`
- `POST /api/v1/brand-portal/brands/:brandId/members`
- `GET /api/v1/brand-portal/brands/:brandId/members`
- `GET /api/v1/brand-portal/brands/:brandId/customers`
- `POST /api/v1/brand-portal/brands/:brandId/customers/offline-purchase`
- `POST /api/v1/admin/brands`
- `PATCH /api/v1/admin/brands/:brandId/status`

## Behavior notes

- Không thêm global role `BRAND`; global role vẫn là `USER | ADMIN`.
- Brand staff vẫn là `users`, quyền brand nằm ở `brand_members`.
- USER authenticated tạo brand sẽ tạo `brands.status = PENDING_REVIEW` và owner `brand_members.status = ACTIVE`.
- ADMIN có thể tạo brand `ACTIVE` trực tiếp.
- ADMIN approve brand bằng cách update status sang `ACTIVE`; khi đó set `approved_by_user_id` và `approved_at`.
- Brand portal actions dùng helper tập trung `RequireBrandRole`.
- Brand portal yêu cầu brand `ACTIVE`, member `ACTIVE`, và role phù hợp.
- Offline customer tạo trong `brand_customers` với `user_id = NULL`, không tạo `users`.
- Offline customer dedupe theo `brand_id + phone_hash`.

## Tests added/updated

- Chưa thêm unit test riêng cho brand rules ở sub-phase này.
- Đã chạy regression/compile:
  - `go test ./...`: pass.
  - `make wire`: pass.
  - `make swagger`: pass.
  - `make build`: pass.

## Backward compatibility notes

- Không thay đổi global `users.role_slug`.
- Không chạm auth hiện tại.
- Không tạo campaign, brand subscription, brand orders.
- `POST /brands/:brandId/join-loyalty` hiện chỉ tạo `brand_customers`; loyalty account sẽ hoàn thiện ở Phase 05b.

## Manual verification steps

1. Chạy migration trong DB test/staging.
2. USER gọi `POST /api/v1/brand-portal/brands`, kiểm tra brand ở `PENDING_REVIEW` và có owner member.
3. Gọi `GET /api/v1/brands`, kiểm tra brand pending chưa public.
4. ADMIN gọi `PATCH /api/v1/admin/brands/:brandId/status` với `ACTIVE`.
5. Gọi lại `GET /api/v1/brands`, kiểm tra brand active đã public.
6. USER gọi `POST /api/v1/brands/:brandId/join-loyalty`, kiểm tra tạo `brand_customers` với `joined_source = SELF_JOIN`.
7. Brand staff gọi offline purchase endpoint, kiểm tra `brand_customers.user_id IS NULL` và `joined_source = OFFLINE_PURCHASE`.

## Known limitations

- Chưa có loyalty account vì thuộc Phase 05b.
- Chưa có granular permission matrix sâu hơn role helper hiện tại.
- Chưa có unit/handler test riêng cho brand module.
- Chưa chạy `make migration-up`.
