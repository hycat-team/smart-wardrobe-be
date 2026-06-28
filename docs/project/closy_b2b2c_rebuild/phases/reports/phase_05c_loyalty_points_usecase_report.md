# Phase 05c - Unified Loyalty Points Usecase

## Trạng thái

Đã hoàn thành API thống nhất để brand staff ghi nhận điểm loyalty bằng `userId`, `phone` hoặc `externalCustomerCode`. Không có migration mới trong sub-phase này.

## Files changed

- `internal/modules/brand/application/dto/brand.go`
- `internal/modules/brand/application/errors/errors.go`
- `internal/modules/brand/application/interface/usecase/brand_core_uc.go`
- `internal/modules/brand/application/usecase/brand_core_uc.go`
- `internal/modules/brand/domain/repositories/interfaces.go`
- `internal/modules/brand/infrastructure/persistence/brand_repo.go`
- `internal/modules/brand/infrastructure/persistence/loyalty_repo.go`
- `internal/modules/brand/presentation/handler/brand_handler.go`
- `internal/api/routes/brand/router.go`
- `internal/di/wire_gen.go`
- `api/swagger/docs.go`
- `api/swagger/swagger.json`
- `api/swagger/swagger.yaml`

## Migrations added

- Không có migration mới.
- Không chạy `migrate up`.

## APIs added/changed

- Thêm `POST /api/v1/brand-portal/brands/:brandId/loyalty/points`

Request hỗ trợ:

- `userId`
- `phone`
- `externalCustomerCode`
- `purchaseAmount`
- `pointsDelta`
- `transactionType`
- `reason`
- `referenceType`
- `referenceId`
- `idempotencyKey`

## Code behavior

- Chỉ brand member role `OWNER` hoặc `MANAGER` được ghi nhận điểm.
- Endpoint chỉ cho phép transaction type `EARN`, `ADJUST`, `REFUND`.
- `REDEEM` và `EXPIRE` không mở qua endpoint này.
- Nếu có `userId`, backend tìm hoặc tạo `brand_customers` linked với user thật.
- Nếu chỉ có `phone`, backend tạo hoặc dùng lại `brand_customers` offline với `user_id = NULL`, không tạo `users`.
- Nếu chỉ có `externalCustomerCode`, backend tạo hoặc dùng lại offline `brand_customers`.
- `joined_source` dùng `SELF_JOIN` hoặc `OFFLINE_PURCHASE`, không dùng `STAFF_CREATED`.
- Mọi thay đổi balance chạy trong DB transaction.
- `loyalty_accounts` được lock bằng `FOR UPDATE` trước khi tính balance mới.
- Nếu có `idempotencyKey` trùng trong cùng brand, backend trả transaction/result cũ và không cộng điểm lần hai.
- Nếu `purchaseAmount` có và `pointsDelta` null, backend tính điểm theo active `loyalty_programs.amount_per_point` và `rounding_mode`.
- Nếu purchase nhỏ ra 0 điểm, backend vẫn cập nhật `total_spend` và tier, nhưng không insert transaction 0 điểm.
- `loyalty_point_transactions` chỉ insert append-only, không update transaction cũ.
- Không dùng `remaining_points`.

## Tests added/updated

- Chưa thêm unit test riêng cho loyalty point rules.
- Đã chạy regression/compile:
  - `go test ./...`: pass.
  - `make wire`: pass.
  - `make swagger`: pass.
  - `make build`: pass.

## Backward compatibility notes

- Không đổi auth hiện tại.
- Không tạo endpoint offline purchase riêng.
- Không tạo `brand_orders`.
- Không tạo `users` từ phone offline.
- Không dùng SMS/phone OTP.

## Manual verification steps

1. Chạy migrations 05a và 05b trong DB test/staging.
2. Tạo brand `ACTIVE`, owner/manager member và active loyalty program.
3. Gọi endpoint bằng `phone` chưa có customer, xác nhận:
   - tạo `brand_customers.user_id IS NULL`
   - tạo `loyalty_accounts`
   - insert `loyalty_point_transactions`
4. Gọi lại cùng `idempotencyKey`, xác nhận không cộng điểm lần hai.
5. Gọi endpoint bằng `userId`, xác nhận không cần phone.
6. Thử `REDEEM` hoặc `EXPIRE`, xác nhận bị từ chối.
7. Thử cộng âm làm balance dưới 0, xác nhận bị từ chối.

## Known limitations

- Chưa có unit test riêng cho FLOOR/ROUND/CEIL và idempotency.
- Chưa có endpoint quản lý loyalty program/tier; phase này dùng schema/repo đã có.
- Không xử lý purchase refund giảm `total_spend` vì chưa có `brand_orders`.
