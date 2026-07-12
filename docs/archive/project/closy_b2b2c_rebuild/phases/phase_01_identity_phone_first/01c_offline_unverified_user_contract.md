# Phase 01c - Không tạo offline unverified users

## Mục tiêu

Thay thế contract cũ `CreateBrandCreatedUnverifiedUser` bằng quyết định mới: brand không tạo tài khoản Closy thay khách offline trong MVP.

## Không làm trong phase này

```text
- Không tạo contract identity để find-or-create user bằng phone.
- Không tạo user UNVERIFIED.
- Không set registration_source = BRAND_CREATED.
- Không activate user bằng phone OTP.
```

## Contract đúng trong MVP

Module `identity` chỉ nên cung cấp các contract đọc user thật hiện có, ví dụ:

```text
GetUserByID
GetUserByEmailOrUsername
```

Module `brand` chịu trách nhiệm offline loyalty qua:

```text
CreateOrResolveOfflineBrandCustomer(phone/customer info)
GrantOrAdjustLoyaltyPoints(input)
CreateBrandCustomerClaim(brandCustomerID)
ClaimBrandCustomer(currentUserID, claimToken)
```

## Linked/unlinked rule

Không dùng `link_status`.

```text
brand_customers.user_id IS NULL     = offline/unlinked customer
brand_customers.user_id IS NOT NULL = linked Closy user
```

## Tests

- Brand offline purchase bằng phone không tạo row trong `users`.
- Brand offline purchase tạo hoặc dùng lại `brand_customers` theo `brand_id + phone_hash`.
- Claim token hợp lệ link `brand_customers.user_id` với current user.

## Acceptance checklist

- [ ] Không có contract tạo `users UNVERIFIED`.
- [ ] Offline identity nằm trong `brand_customers`.
- [ ] Claim/link dùng claim token/QR, không dùng phone OTP.
