# Phase 01a - Users Schema for Phone-First Identity

## Mục tiêu

Mở rộng bảng `users` để hỗ trợ phone-first identity và offline loyalty acquisition, đồng thời giữ backward compatibility cho user cũ đăng ký bằng email.

## Không làm trong phase này

```text
- Không tạo bảng OTP.
- Không đổi toàn bộ auth flow ngay.
- Không xóa email.
- Không ép user cũ phải có phone ở cấp DB.
- Không tạo brand/loyalty tables trong phase này.
```

## Target behavior

User mới self-signup sau rebuild phải có phone.

User cũ có email nhưng chưa có phone vẫn tồn tại được trong DB.

User do brand tạo offline:

```text
status = UNVERIFIED
registration_source = BRAND_CREATED
phone_e164 = số điện thoại đã normalize
phone_verified_at = NULL
```

## Migration target

Thêm hoặc điều chỉnh field trong `users`:

```text
phone_e164 nullable unique
phone_verified_at nullable
email nullable unique
email_verified_at nullable
display_name nullable
status: UNVERIFIED | ACTIVE | SUSPENDED | DELETED
registration_source: SELF_SIGNUP | BRAND_CREATED | ADMIN_CREATED
created_at
updated_at
```

Nếu `display_name`, `status`, `created_at`, `updated_at` đã tồn tại thì preserve.

Nếu `email` hiện đang `NOT NULL`, cần migration đổi sang nullable để support brand-created phone-only user.

Nếu `password_hash` hiện đang `NOT NULL`, có 2 hướng. Agent phải chọn theo current auth design:

```text
A. Nếu auth đang chuyển sang OTP-only phone login:
   - cho phép password_hash nullable.

B. Nếu auth vẫn cần password_hash:
   - với user BRAND_CREATED/UNVERIFIED, set password_hash thành unusable hash/sentinel theo convention hiện có.
   - tuyệt đối không cho login bằng password khi status UNVERIFIED.
```

Không tự chọn nếu chưa xác minh current auth flow.

## Index rules

Postgres cho phép nhiều NULL trong UNIQUE, nhưng nên dùng partial unique index rõ ràng nếu phù hợp:

```sql
CREATE UNIQUE INDEX IF NOT EXISTS users_phone_e164_unique
ON users(phone_e164)
WHERE phone_e164 IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS users_email_unique
ON users(email)
WHERE email IS NOT NULL;
```

Nếu đã có unique constraint trên email, phải migration cẩn thận, tránh duplicate index conflict.

## Entity updates

Cập nhật user entity/model/DTO để có:

```text
PhoneE164
PhoneVerifiedAt
Email nullable
EmailVerifiedAt nullable
Status
RegistrationSource
DisplayName nullable
```

Status enum:

```text
UNVERIFIED
ACTIVE
SUSPENDED
DELETED
```

Registration source enum:

```text
SELF_SIGNUP
BRAND_CREATED
ADMIN_CREATED
```

Nếu code hiện dùng numeric enum, không tự đổi sang string nếu làm vỡ current system. Có thể map string concept vào enum hiện tại.

## Phone normalization

Tạo hoặc dùng utility hiện có để normalize phone thành E.164.

Rule MVP:

```text
- Lưu DB bằng phone_e164 đã normalize.
- Không lưu raw phone làm khóa chính.
- Nếu input là số Việt Nam dạng 09..., normalize thành +849...
- Nếu input đã có +84, giữ đúng E.164.
- Validate format trước khi insert.
```

Nếu repo đã có phone normalization utility thì dùng lại.

## Backward compatibility

User cũ email-only:

```text
phone_e164 = NULL
phone_verified_at = NULL
email giữ nguyên
registration_source default SELF_SIGNUP nếu không xác định được
status giữ ACTIVE nếu đang active
```

Tài khoản mới sau phase này phải đi qua validation ở usecase, không dựa vào DB NOT NULL cho phone vì DB cần backward compatible.

## Tests

Migration/integration tests:

- User cũ email-only vẫn tồn tại sau migration.
- Có thể tạo user phone-only với email NULL.
- Không thể tạo 2 users cùng `phone_e164` non-null.
- Có thể có nhiều users với email NULL.
- `BRAND_CREATED` user có `UNVERIFIED` và `phone_verified_at NULL`.

Unit tests:

- Normalize `0901234567` -> `+84901234567`.
- Reject phone invalid.
- Preserve existing `+84...`.

## Acceptance checklist

- [ ] Migration chạy được trên DB hiện tại.
- [ ] Không tạo bảng OTP.
- [ ] Email nullable nhưng unique khi non-null.
- [ ] Phone nullable nhưng unique khi non-null.
- [ ] User cũ không bị mất khả năng login.
- [ ] User brand-created unverified có thể được tạo sau phase này.

## Lỗi cần tránh

- Đổi `email` thành optional trong DB nhưng quên update validation ở DTO.
- Ép `phone_e164 NOT NULL` làm fail user cũ.
- Tạo bảng `phone_otp_challenges` trái scope.
- Cho user `UNVERIFIED` login đầy đủ.
