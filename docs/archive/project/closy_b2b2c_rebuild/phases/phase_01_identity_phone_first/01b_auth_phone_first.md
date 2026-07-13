# Phase 01b - Giữ nguyên auth hiện tại

## Mục tiêu

Xác nhận auth hiện tại của Closy là auth chính trong MVP B2B2C. Không mở rộng sang phone login/phone OTP ở giai đoạn này.

## Không làm trong phase này

```text
- Không đổi register sang phone-first.
- Không đổi login sang phone/password hoặc phone OTP.
- Không đổi forgot-password sang phone OTP.
- Không thêm SMS/Zalo OTP provider.
- Không tạo endpoint claim offline loyalty trong auth module.
```

## Auth behavior giữ nguyên

Các endpoint auth hiện tại tiếp tục là nguồn đăng nhập chính:

```text
POST /api/v1/auth/register
POST /api/v1/auth/register/confirm-otp
POST /api/v1/auth/register/resend-otp
POST /api/v1/auth/login
POST /api/v1/auth/refresh-token
POST /api/v1/auth/forgot-password
POST /api/v1/auth/forgot-password/confirm-otp
POST /api/v1/auth/forgot-password/resend-otp
POST /api/v1/auth/reset-password
POST /api/v1/auth/logout
```

OTP hiện tại vẫn lưu Redis theo implementation hiện có. Không tạo bảng OTP.

## Offline loyalty claim không nằm trong auth

Claim/link offline loyalty dùng claim token/QR qua brand module:

```text
brand_customer_claims
brand_customers.user_id nullable -> set current_user.id khi claim thành công
loyalty_accounts.user_id nullable -> set current_user.id khi claim thành công
```

Auth chỉ cung cấp current user đã đăng nhập để brand module thực hiện claim.

## Tests

- Register email/password + email OTP vẫn pass.
- Login username/email + password vẫn pass.
- Forgot/reset password email OTP vẫn pass.
- Không có endpoint auth phone OTP mới.

## Acceptance checklist

- [ ] Không thay đổi request/response auth hiện tại.
- [ ] Không thêm SMS provider.
- [ ] Không tạo OTP DB table.
- [ ] Claim/link offline loyalty được defer sang Phase 05 brand module.
