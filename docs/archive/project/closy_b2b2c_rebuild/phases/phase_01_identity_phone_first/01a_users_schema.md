# Phase 01a - Giữ nguyên users schema và auth hiện tại

## Mục tiêu

Cập nhật định hướng Phase 01 theo quyết định mới: MVP không chuyển sang phone-first identity. Phase này chỉ xác nhận và bảo vệ auth/schema hiện tại trước khi đi tiếp sang các phase B2B2C khác.

## Không làm trong phase này

```text
- Không thêm phone_e164 vào users để làm định danh chính.
- Không ép email nullable.
- Không ép password_hash nullable.
- Không đổi status users sang UNVERIFIED/ACTIVE dạng string.
- Không thêm registration_source vào users cho offline customer.
- Không tạo users UNVERIFIED từ brand offline purchase.
- Không tạo phone_otp_challenges.
- Không tích hợp SMS OTP/Zalo OTP.
```

## Quyết định schema

Trong MVP, `users` vẫn là tài khoản thật của người dùng Closy app.

Giữ backward compatibility với schema hiện tại:

```text
email NOT NULL
password_hash NOT NULL
status SMALLINT theo enum hiện tại
```

Nếu sau này cần bổ sung phone vào hồ sơ user, việc đó phải là phase riêng và không được biến phone thành dependency bắt buộc cho MVP.

## Offline customer không nằm trong users

Khách offline mua hàng tại brand nhưng chưa chủ động dùng Closy không được tạo thành `users`.

Thay vào đó, Phase 05 dùng:

```text
brand_customers.user_id nullable
brand_customers.phone_hash nullable
brand_customers.phone_e164 nullable nếu cần visibility nội bộ theo brand
brand_customer_claims để link tài khoản bằng claim code/QR
```

## Tests

- Auth register/login hiện tại vẫn hoạt động.
- Không có migration Phase 01 nào làm đổi nullability của `users.email` hoặc `users.password_hash`.
- Không có table `phone_otp_challenges`.
- Không có code path tạo `users UNVERIFIED` từ brand offline purchase.

## Acceptance checklist

- [ ] Auth hiện tại được giữ nguyên.
- [ ] Không thêm phone-first requirement cho user mới.
- [ ] Không tạo offline user trong `users`.
- [ ] Offline loyalty được chuyển sang Phase 05 `brand_customers`.
- [ ] Chưa thay đổi behavior production.
