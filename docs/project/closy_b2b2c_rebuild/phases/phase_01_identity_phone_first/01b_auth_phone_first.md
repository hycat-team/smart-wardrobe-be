# Phase 01b - Auth Flow Phone-First Extension

## Mục tiêu

Mở rộng auth để tài khoản mới dùng phone-first, vẫn giữ backward compatibility cho email user cũ. OTP tiếp tục dùng Redis flow hiện có.

## Không làm trong phase này

```text
- Không tạo bảng OTP.
- Không viết SMS provider mới nếu hệ thống đã có OTP provider.
- Không xóa email login của user cũ ngay.
- Không tạo brand loyalty flow trong phase này.
```

## Signup behavior

Self-signup mới:

```text
input: phone, optional display_name, optional email nếu app có
normalize phone -> phone_e164
create user nếu phone chưa tồn tại
send/verify OTP bằng Redis flow hiện có
sau verify: status ACTIVE, phone_verified_at now
registration_source SELF_SIGNUP
```

Nếu current auth vẫn password-based, agent phải giữ behavior password hiện có nhưng phone là identity chính cho user mới.

## Login behavior

MVP behavior chấp nhận:

```text
- Phone login dùng OTP/password theo flow hiện có sau khi phone verified.
- Email login giữ cho user cũ nếu email_verified_at hoặc legacy account hợp lệ.
- User UNVERIFIED không được login đầy đủ.
```

Nếu user nhập phone trùng một `UNVERIFIED` user do brand tạo:

```text
- Gửi OTP theo Redis flow hiện có.
- Sau verify, mark user ACTIVE.
- Không tạo user mới.
```

## Forgot password / account recovery

Mở rộng endpoint hiện có, không tạo endpoint song song nếu đã có:

```text
POST /api/v1/auth/forgot-password
POST /api/v1/auth/forgot-password/confirm-otp
POST /api/v1/auth/forgot-password/resend-otp
POST /api/v1/auth/reset-password
```

Yêu cầu bảo mật:

```text
- Nếu recovery bằng phone, chỉ cho phép nếu phone_verified_at IS NOT NULL.
- Nếu recovery bằng email, chỉ cho phép nếu email_verified_at IS NOT NULL.
- Không cho recovery bằng identifier chưa verify.
- Response nên tránh lộ identifier có tồn tại hay không nếu current system đã theo security pattern này.
```

## Email update after login

Email optional. Nếu user cập nhật email:

```text
POST /api/v1/me/email/request-update
POST /api/v1/me/email/verify-update
```

Rule:

```text
- Chỉ user ACTIVE mới update email.
- Email mới phải unique nếu non-null.
- Gửi OTP qua email bằng flow hiện có nếu có.
- Chỉ set users.email và email_verified_at sau khi verify thành công.
```

Nếu hệ thống chưa có email OTP, phase có thể ghi TODO và không implement email update ngay, nhưng không được set `email_verified_at` khi chưa verify.

## Handler/route update

Cập nhật Swagger nếu repo dùng `swaggo/swag`.

Cập nhật request DTO để hỗ trợ:

```text
phone
phone_e164 nội bộ sau normalization
email nullable
```

Không expose `registration_source` cho client tự set, trừ admin/brand portal usecase sau này.

## Tests

- Self-signup bằng phone mới tạo user ACTIVE sau verify.
- User brand-created UNVERIFIED claim account bằng phone OTP và chuyển ACTIVE.
- UNVERIFIED user không login đầy đủ trước verify.
- Forgot password bằng unverified phone bị từ chối.
- Forgot password bằng verified phone chạy flow hiện có.
- Email update không set email trước khi verify.

## Acceptance checklist

- [ ] Không tạo DB OTP table.
- [ ] Auth mới không phá user cũ email-only.
- [ ] Phone là identity chính cho user mới.
- [ ] Brand-created user có thể claim bằng phone.
- [ ] Swagger/docs cập nhật nếu có.
