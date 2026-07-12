# Báo cáo Phase 01 - Giữ nguyên current auth

## File đã thay đổi

- `docs/project/closy_b2b2c_rebuild/phases/reports/phase_01_current_auth_alignment_report.md`

## Migration đã thêm

- Không có.

## API đã thêm/thay đổi

- Không có API thật nào được thêm hoặc sửa.
- Phase 01 xác nhận giữ nguyên các endpoint auth hiện tại:
  - `POST /api/v1/auth/register`
  - `POST /api/v1/auth/register/confirm-otp`
  - `POST /api/v1/auth/register/resend-otp`
  - `POST /api/v1/auth/login`
  - `POST /api/v1/auth/refresh-token`
  - `POST /api/v1/auth/forgot-password`
  - `POST /api/v1/auth/forgot-password/confirm-otp`
  - `POST /api/v1/auth/forgot-password/resend-otp`
  - `POST /api/v1/auth/reset-password`
  - `POST /api/v1/auth/logout`

## Test đã thêm/cập nhật

- Không có test code mới.
- Không chạy test/build vì Phase 01 theo quyết định mới là phase alignment tài liệu, không thay đổi code production.

## Ghi chú tương thích ngược

- Auth hiện tại của Closy được giữ nguyên: email/username + password, OTP email lưu Redis cho register/forgot-password.
- Không chuyển sang phone-first identity trong MVP.
- Không đổi nullability của `users.email` hoặc `users.password_hash`.
- Không tạo `phone_otp_challenges`.
- Không tạo `users UNVERIFIED` từ brand offline purchase.
- Offline loyalty được defer sang Phase 05 qua `brand_customers.user_id NULL` và `brand_customer_claims`.

## Bước kiểm tra thủ công

- Đọc lại `01a_users_schema.md`, `01b_auth_phone_first.md`, `01c_offline_unverified_user_contract.md`.
- Xác nhận nội dung Phase 01 hiện là giữ current auth, dù folder/file vẫn giữ tên cũ để tránh đổi path rộng.
- Quét các shared rules và README phase để đảm bảo quyết định mới đã được phản ánh.
- Xác nhận không cần câu hỏi thêm trước khi sang Phase 02.

## Giới hạn đã biết

- Tên folder `phase_01_identity_phone_first` và một số tên file cũ vẫn còn để tránh đổi đường dẫn rộng; nội dung bên trong đã được cập nhật theo quyết định mới.
- Chưa cập nhật toàn bộ tài liệu lịch sử ngoài bộ `phases/` và report rebuild chính.
- Chưa code, chưa migration, chưa chạy test/build.

