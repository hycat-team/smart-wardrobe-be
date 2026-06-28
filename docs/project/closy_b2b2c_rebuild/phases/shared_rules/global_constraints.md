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

## Phone-first identity

- `phone_e164` là định danh chính cho tài khoản mới.
- `email` là optional.
- OTP không tạo bảng DB vì hệ thống đang dùng Redis OTP hiện có.
- User cũ đăng ký bằng email vẫn được giữ backward compatibility.
- User mới self-signup phải có phone.
- User offline do brand tạo bằng phone có `status = UNVERIFIED` và `registration_source = BRAND_CREATED`.

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
