# Báo cáo Phase 05c2 - Loyalty Point Lots & Expiry-Safe Redeem

## Kết quả

- Đã thêm migration `20260628145000_create_loyalty_point_lots.sql` để tạo bảng `loyalty_point_lots`, index vận hành expiry/redeem và backfill đơn giản từ các transaction `EARN` hiện có.
- Đã thêm domain constant `loyaltypointlotstatus` với các trạng thái `ACTIVE`, `CONSUMED`, `EXPIRED`.
- Đã thêm entity `LoyaltyPointLot` và mapping table name `loyalty_point_lots`.
- Đã thêm repository lots với các method lock row bằng `FOR UPDATE`:
  - `ListRedeemableLotsForUpdate`
  - `ListExpiredLotsForUpdate`
  - `UpdateLotRemainingAndStatus`
  - `ListAccountsWithExpiredLots`
- Đã cập nhật luồng `GrantLoyaltyPoints` để transaction `EARN` dương tạo thêm một point lot trong cùng DB transaction.
- Đã siết staff endpoint MVP chỉ cho phép `EARN`; `ADJUST`, `REFUND`, `EXPIRE`, `REDEEM` không expose qua endpoint staff.
- Đã thêm helper nội bộ:
  - `expireDueLotsForAccount`
  - `redeemLoyaltyPointsFromLots`
  - `ProcessExpiredLoyaltyPointLots`
- Đã thêm `LoyaltyPointExpiryWorker` trong brand module.
- Đã đăng ký worker vào bootstrap `AppWorkers`; worker chạy một lần lúc startup và lặp theo interval nội bộ.
- Đã thêm config/env:
  - `LOYALTY_EXPIRY_WORKER_ENABLED`
  - `LOYALTY_EXPIRY_WORKER_INTERVAL`
  - `LOYALTY_EXPIRY_WORKER_BATCH_SIZE`

## Ghi chú triển khai

- Không chạy `migrate up`.
- Không thêm `remaining_points` vào `loyalty_point_transactions`.
- Không update transaction cũ; ledger vẫn append-only.
- Worker xử lý theo batch account, mỗi account chạy trong DB transaction riêng.
- Worker idempotent vì chỉ query lots `ACTIVE`, `remaining_points > 0`, `expires_at <= now`; lots đã `EXPIRED` hoặc `remaining_points = 0` sẽ bị bỏ qua.
- Mỗi account gom nhiều expired lots thành một transaction `EXPIRE`.
- Expire/redeem không giảm `lifetime_points`, không giảm `total_spend`, không đổi `current_tier_id`.
- Scheduled worker không thay thế expiry on-demand: redeem helper vẫn gọi `expireDueLotsForAccount` trước khi check đủ điểm.
- Chưa sửa benefit redemption vì endpoint/schema `brand_benefits` và `benefit_redemptions` chưa được triển khai trong code hiện tại; helper redeem từ lots đã sẵn sàng để phase 05d gọi trong cùng transaction.

## Verification

- Test worker/service expire nhiều lots cùng account tạo một transaction `EXPIRE`: pass.
- Test worker chạy lại không trừ điểm lần hai: pass.
- Test redeem sau khi có expired lot thì expire trước rồi mới check điểm: pass.
- `make wire`: pass.
- `go test ./...`: pass.
- `make build`: pass.
