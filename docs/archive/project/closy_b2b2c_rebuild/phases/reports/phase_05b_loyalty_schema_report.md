# Phase 05b - Loyalty Schema

## Trạng thái

Đã hoàn thành schema loyalty tối thiểu, domain entities và repository nền cho module `brand`. Chưa chạy `migrate up`.

## Files changed

- `migrations/20260628132100_loyalty_schema.sql`
- `internal/shared/domain/entities/loyalty_entities.go`
- `internal/shared/domain/entities/table_name.go`
- `internal/shared/domain/constants/loyaltyroundingmode/loyalty_rounding_mode.go`
- `internal/shared/domain/constants/loyaltytransactiontype/loyalty_transaction_type.go`
- `internal/modules/brand/domain/repositories/interfaces.go`
- `internal/modules/brand/infrastructure/persistence/loyalty_repo.go`
- `internal/modules/brand/application/usecase/brand_core_uc.go`
- `internal/modules/brand/provider.go`
- `internal/di/wire_gen.go`

## Migrations added

- `migrations/20260628132100_loyalty_schema.sql`

Migration tạo:

- `loyalty_programs`
- `loyalty_tiers`
- `loyalty_accounts`
- `loyalty_point_transactions`
- `brand_customer_claims`

Các constraint/index chính:

- Chỉ một active `loyalty_programs` mỗi brand bằng partial unique index.
- `loyalty_accounts.brand_customer_id` unique.
- `loyalty_accounts(brand_id, user_id)` unique khi `user_id IS NOT NULL`.
- `loyalty_point_transactions(brand_id, idempotency_key)` unique khi `idempotency_key IS NOT NULL`.
- Index đọc ledger theo account/customer/time.
- Index `expires_at` cho transaction có hạn dùng.

## APIs added/changed

- Không thêm endpoint mới trong 05b.
- Cập nhật behavior `POST /api/v1/brands/:brandId/join-loyalty`: sau khi tạo hoặc lấy `brand_customers`, backend bảo đảm có `loyalty_accounts`.
- Cập nhật offline customer flow trong brand portal: sau khi tạo hoặc lấy offline `brand_customers`, backend bảo đảm có `loyalty_accounts`.

## Code behavior

- `brand_customer_id` là identity chính của loyalty account.
- `user_id` trong `loyalty_accounts` nullable để hỗ trợ offline customer chưa link Closy account.
- `loyalty_point_transactions` là ledger append-only ở tầng schema/domain.
- Không có `remaining_points`.
- Không có `loyalty_point_lots`.
- Tier dùng `loyalty_accounts.total_spend` và `loyalty_tiers.min_total_spend`, không dùng `current_points`.
- `brand_customer_claims.claim_token_hash` lưu hash token, không lưu raw token.

## Tests added/updated

- Chưa thêm unit/integration test riêng cho schema constraints vì repo hiện chưa có DB integration test harness.
- Đã chạy regression/compile:
  - `go test ./...`: pass.
  - `make wire`: pass.
  - `make build`: pass.

## Backward compatibility notes

- Không thay đổi auth hoặc global user role.
- Không tạo `brand_orders`.
- Không tạo `brand_order_items`.
- Không tạo `campaign`.
- Không tạo `loyalty_point_lots`.
- Không thêm `remaining_points` vào ledger.

## Manual verification steps

1. Chạy migration trong DB test/staging.
2. Kiểm tra chỉ một active program mỗi brand:
   - tạo 2 `loyalty_programs` active cùng `brand_id` phải fail.
3. Kiểm tra unique account:
   - tạo 2 `loyalty_accounts` cùng `brand_customer_id` phải fail.
   - tạo 2 `loyalty_accounts` cùng `(brand_id, user_id)` khi `user_id` non-null phải fail.
4. Kiểm tra idempotency:
   - tạo 2 transaction cùng `(brand_id, idempotency_key)` non-null phải fail.
5. Kiểm tra không có cột/bảng cấm:
   - không có `loyalty_point_lots`.
   - không có `remaining_points`.
6. Gọi join loyalty sau khi apply Phase 05a + 05b migrations, xác nhận có `brand_customers` và `loyalty_accounts`.

## Known limitations

- Chưa có API grant/adjust/redeem điểm; phần đó thuộc Phase 05c.
- Chưa có expiry job; schema đã hỗ trợ `expires_at` trên EARN transaction.
- Chưa có claim token usecase; schema `brand_customer_claims` đã sẵn sàng cho phase sau.
- Chưa chạy `make migration-up`.
