# Phase 01c - Offline Unverified User Contract

## Mục tiêu

Chuẩn hóa contract để module `brand` có thể tạo hoặc lấy user theo số điện thoại khi brand staff tích điểm cho khách offline.

Phase này chỉ chuẩn bị identity contract. Loyalty usecase triển khai ở Phase 05.

## Không làm trong phase này

```text
- Không tạo brand tables.
- Không cộng điểm.
- Không tạo loyalty account.
- Không tạo phone OTP table.
```

## Contract cần có trong identity

Tên function có thể khác theo codebase, nhưng behavior phải tương đương:

```text
FindUserByPhone(phoneE164) -> UserDTO | nil
CreateBrandCreatedUnverifiedUser(input) -> UserDTO
FindOrCreateBrandCreatedUserByPhone(input) -> UserDTO
MarkPhoneVerified(userID, phoneE164) -> UserDTO
```

Input cho create/find-or-create:

```text
phone raw hoặc phone_e164
customer_name/display_name nullable
created_by_user_id nullable
source = BRAND_CREATED
```

## Behavior bắt buộc

`FindOrCreateBrandCreatedUserByPhone`:

```text
- normalize phone về E.164
- nếu user tồn tại theo phone_e164: return user đó
- nếu chưa tồn tại: create user UNVERIFIED, registration_source BRAND_CREATED
- không gửi OTP trong function này
- không activate account trong function này
- không cho brand set password cho user
```

Nếu user tồn tại nhưng `status = DELETED` hoặc `SUSPENDED`, không tự reactivate. Trả lỗi domain rõ ràng để brand usecase xử lý.

## Security rule

Brand staff chỉ được tạo offline user thông qua brand loyalty usecase sau khi đã pass `brand_members` permission.

Identity contract không tự check brand permission, vì identity không sở hữu brand tables. Brand module phải check trước khi gọi.

## Tests

- Phone chưa tồn tại -> tạo UNVERIFIED/BRAND_CREATED.
- Phone đã tồn tại ACTIVE -> return user hiện có.
- Phone đã tồn tại UNVERIFIED -> return user hiện có, không tạo duplicate.
- Phone invalid -> lỗi validation.
- User SUSPENDED/DELETED -> lỗi, không tạo user mới cùng phone.

## Acceptance checklist

- [ ] Có contract hoặc service method cho brand dùng.
- [ ] Không có duplicate phone user.
- [ ] Không có OTP table.
- [ ] Không activate user khi brand tạo offline.
