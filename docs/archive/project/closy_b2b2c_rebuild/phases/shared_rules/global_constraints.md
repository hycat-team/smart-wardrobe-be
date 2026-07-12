# Global Constraints

## Runtime modules được phép tồn tại

Chỉ dùng 5 runtime modules:

```text
identity
subscription
wardrobe
styling
brand
```

Không tạo thêm runtime module riêng cho:

```text
garment
samplelab
campaign
loyalty
chat
brand_subscription
```

Nếu cần package con bên trong `brand`, có thể tạo folder nội bộ như `brand/domain/loyalty`, `brand/domain/chat`, nhưng không được biến thành module runtime độc lập.

## Ngoài scope MVP

Không làm trong MVP rebuild phase hiện tại:

```text
phone-first identity
create UNVERIFIED users from brand offline purchase
user_brand_consents
phone_otp_challenges
loyalty_point_lots
brand_orders
brand_order_items
support_tickets
return_exchange_requests
campaign
brand subscription / B2B billing
complex loyalty rule engine
multiple active loyalty programs per brand
rolling yearly tier reset
required_brand_item_id
GenerateSampleTrialStyling
GenerateDigitalSampleOutfit
sample_outfit_trials
sample_trial_items
digital_sample_variants
garment_specs
```

## Identity trong MVP

- Không chuyển sang phone-first identity trong MVP.
- Giữ auth hiện tại của Closy, gồm đăng ký/đăng nhập bằng email hoặc username + password và OTP email theo Redis flow hiện có.
- Không bắt buộc user mới phải đăng ký bằng số điện thoại.
- Không tạo `users` cho khách offline khi brand nhập số điện thoại tại cửa hàng.
- Brand staff vẫn là `users` thật, được cấp quyền quản trị brand qua `brand_members`.
- Nếu sau này cần phone/SMS/Zalo OTP, chỉ thêm implementation hạ tầng sau khi business chốt provider; MVP không phụ thuộc SMS OTP.

## Offline loyalty identity

- Khách offline chưa có tài khoản Closy được lưu trong `brand_customers`, không được tạo thành `users`.
- `brand_customers.user_id IS NULL` nghĩa là offline/unlinked customer.
- `brand_customers.user_id IS NOT NULL` nghĩa là đã linked với tài khoản Closy thật.
- Không thêm `link_status`; trạng thái linked/unlinked suy ra từ `user_id`.
- `phone_hash` dùng để lookup/dedupe theo số điện thoại mà không bắt buộc phụ thuộc raw phone.
- Claim/link account dùng claim code hoặc QR claim token, không dùng OTP phone trong MVP.

## Brand identity

Brand không phải account login riêng.

```text
users = người thật đăng nhập
brands = tổ chức / nhãn hàng
brand_members = user nào có quyền quản trị brand nào
```

Brand Portal access luôn phải check:

```text
brands.status = ACTIVE
brand_members.status = ACTIVE
brand_members.role hợp lệ với action
```

## Item model

```text
fashion_items = item lõi: image, metadata, embedding, AI processing state
wardrobe_items = wrapper đồ thuộc user
brand_items = wrapper sản phẩm hoặc sample của brand
outfits = outfit của user
outfit_items = item trong outfit, trỏ fashion_item_id và có item_context
```

`outfit_items.item_context` bắt buộc có giá trị:

```text
USER_WARDROBE
BRAND_ITEM
```

## AI outfit + brand items

Không có `required_brand_item_id`.

Input AI outfit recommendation chỉ mở rộng tối thiểu:

```json
{
  "include_brand_items": true
}
```

Nếu `include_brand_items = true`, hệ thống tự tìm brand items hợp lệ qua `brand/contract.ListEligibleBrandItemsForStyling`.

`styling` không tự query brand/loyalty tables.

## Loyalty

Tier dựa trên tổng chi tiêu:

```text
loyalty_accounts.total_spend -> current_tier_id
```

`loyalty_accounts` gắn với `brand_customer_id` là identity chính trong brand. `user_id` nullable để hỗ trợ offline customer chưa linked Closy account.

Points là số dư dùng để redeem benefit:

```text
current_points = projection đọc nhanh
loyalty_point_transactions = ledger bất biến
```

Không được dùng `current_points` để xét tier.

Không được update transaction cũ.

Không có `remaining_points`.

## Privacy

Brand không được xem raw wardrobe của user.

Brand được xem:

```text
- loyalty profile của customer thuộc brand đó
- brand item/sample responses thuộc brand đó
- chat giữa user và brand đó
- outfit interaction có chứa brand_items của brand đó
- aggregate insight không lộ raw wardrobe item cá nhân
```

Brand không được xem:

```text
- toàn bộ wardrobe_items của user
- outfit cá nhân không chứa brand_items của brand
- AI chat riêng của user
- raw personal wardrobe metadata không liên quan brand
```
