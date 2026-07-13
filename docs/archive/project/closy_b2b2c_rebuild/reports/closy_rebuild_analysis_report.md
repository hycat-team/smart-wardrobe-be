# Báo cáo rebuild Closy B2B2C - Quyết định MVP cập nhật

Tài liệu này là bản report self-contained theo quyết định mới: MVP không chuyển sang phone-first identity, không tạo tài khoản Closy thay khách offline, và dùng `brand_customers` làm identity loyalty trong từng brand.

## 1. Quyết định kiến trúc cuối cho MVP

- Chuyển Closy từ B2C wardrobe app sang B2B2C Fashion Loyalty & Co-creation Platform.
- Giữ backend theo hướng modular monolith, tránh over-engineering.
- Chỉ có 5 runtime modules: `identity`, `subscription`, `wardrobe`, `styling`, `brand`.
- Không tạo module runtime riêng cho `garment`, `samplelab`, `campaign`, `loyalty`, `chat`, `brand_subscription`.
- Campaign và brand subscription/B2B billing nằm ngoài scope MVP hiện tại.

## 2. Identity và auth

MVP giữ auth hiện tại của Closy.

Không làm trong MVP:

- Không chuyển sang phone-first identity.
- Không bắt buộc user mới phải đăng ký bằng số điện thoại.
- Không tạo `users UNVERIFIED` khi brand nhập khách offline.
- Không tạo `phone_otp_challenges`.
- Không dùng SMS/phone OTP làm dependency bắt buộc.
- Không thêm `user_brand_consents`.

`users` vẫn là tài khoản thật của người dùng Closy app. Brand staff cũng là `users` hiện có, được cấp quyền quản trị brand thông qua `brand_members`.

Auth hiện tại tiếp tục là nguồn đăng nhập chính:

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

OTP hiện tại vẫn dùng Redis/email theo implementation hiện có.

## 3. Offline loyalty và brand customer identity

Khách offline chưa có tài khoản Closy không được tạo thành `users`.

Thay vào đó:

- Brand tạo hoặc dùng lại hồ sơ khách hàng trong `brand_customers`.
- `brand_customers.user_id IS NULL` nghĩa là khách offline chưa liên kết tài khoản Closy.
- `brand_customers.user_id IS NOT NULL` nghĩa là hồ sơ khách hàng brand đã linked với một tài khoản Closy thật.
- Không thêm `link_status`, vì linked/unlinked được suy ra trực tiếp từ `user_id`.
- Khách offline vẫn có thể được lưu tên, số điện thoại, mã khách hàng, điểm, tier và lịch sử điểm.

Sau này khi khách tự đăng nhập/đăng ký Closy bằng auth hiện tại, khách có thể claim/link hồ sơ loyalty bằng claim code hoặc QR claim token.

## 4. Schema target liên quan brand/loyalty

### `brand_customers`

```text
id UUID PK
brand_id UUID FK brands(id)
user_id UUID FK users(id) NULL
customer_name VARCHAR(255) NULL
phone_e164 VARCHAR(50) NULL
phone_hash VARCHAR(255) NULL
external_customer_code VARCHAR(100) NULL
joined_source VARCHAR(50)
status VARCHAR(50)
joined_at TIMESTAMP
claimed_at TIMESTAMP NULL
created_by_member_id UUID NULL
created_at
updated_at
```

`joined_source` chỉ mô tả nguồn/trigger khiến khách trở thành brand customer:

```text
SELF_JOIN
OFFLINE_PURCHASE
IMPORT optional sau MVP nếu cần migrate danh sách khách cũ
```

MVP có thể chỉ dùng `SELF_JOIN` và `OFFLINE_PURCHASE`. Không dùng `STAFF_CREATED` vì actor tạo record đã được lưu bằng `created_by_member_id`.

Gợi ý unique/index:

```sql
CREATE UNIQUE INDEX ... ON brand_customers(brand_id, user_id)
WHERE user_id IS NOT NULL;

CREATE UNIQUE INDEX ... ON brand_customers(brand_id, phone_hash)
WHERE phone_hash IS NOT NULL;
```

### `brand_customer_claims`

```text
id UUID PK
brand_customer_id UUID FK brand_customers(id)
claim_token_hash VARCHAR(255)
expires_at TIMESTAMP
consumed_at TIMESTAMP NULL
created_at TIMESTAMP
```

Rule:

- Token/code phải lưu dạng hash, không lưu raw token.
- Token dùng một lần và có hạn dùng.
- Chỉ claim nếu `brand_customers.user_id IS NULL`.
- Claim thành công thì set `consumed_at`, `brand_customers.user_id`, `brand_customers.claimed_at`, và `loyalty_accounts.user_id`.
- Không cần OTP phone cho claim trong MVP.

### `loyalty_accounts`

```text
id UUID PK
brand_id UUID FK brands(id)
brand_customer_id UUID FK brand_customers(id)
user_id UUID FK users(id) NULL
current_points INT
lifetime_points INT
total_spend DECIMAL(12,2)
current_tier_id UUID FK loyalty_tiers(id) NULL
created_at
updated_at
```

Rule:

- `brand_customer_id` là identity chính của loyalty trong brand.
- `user_id` nullable để hỗ trợ khách offline.
- Tier dựa trên `total_spend`, không dựa trên `current_points`.
- Redeem điểm không làm giảm tier nếu `total_spend` vẫn đủ.

### `loyalty_point_transactions`

Append-only ledger:

```text
id UUID PK
loyalty_account_id UUID FK loyalty_accounts(id)
brand_id UUID FK brands(id)
brand_customer_id UUID FK brand_customers(id)
user_id UUID FK users(id) NULL
points_delta INT
balance_after INT
transaction_type VARCHAR(50)
reason VARCHAR(255) NULL
spend_amount DECIMAL(12,2) NULL
reference_type VARCHAR(100) NULL
reference_id UUID NULL
expires_at TIMESTAMP NULL
idempotency_key VARCHAR(100) NULL
created_by_user_id UUID FK users(id) NULL
created_at TIMESTAMP
```

Không dùng:

- `remaining_points`
- `loyalty_point_lots`

Mọi thao tác cộng/trừ điểm phải chạy trong DB transaction, lock loyalty account row, update `loyalty_accounts.current_points` atomic cùng transaction ledger, và có idempotency bằng `idempotency_key` hoặc `reference_type + reference_id`.

## 5. Unified loyalty points API

Không tách API offline purchase riêng.

```text
POST /api/v1/brand-portal/brands/:brandId/loyalty/points
```

Request body gợi ý:

```json
{
  "user_id": "nullable",
  "phone": "nullable",
  "customer_name": "nullable",
  "external_customer_code": "nullable",
  "purchase_amount": 500000,
  "points_delta": null,
  "reason": "Offline purchase",
  "reference_type": "MANUAL_PURCHASE",
  "reference_id": "nullable",
  "idempotency_key": "nullable"
}
```

Rule:

- Nếu có `user_id`, tìm hoặc tạo `brand_customer` linked với user đó.
- Nếu không có `user_id` nhưng có `phone`, chuẩn hóa phone, tính `phone_hash`, tìm hoặc tạo `brand_customer` offline/unlinked.
- Không tạo `users` từ phone offline.
- Nếu có `purchase_amount`, tính điểm theo `loyalty_programs.amount_per_point`.
- Nếu có `points_delta`, dùng cho manual adjustment.
- Nếu customer chưa có `loyalty_account`, tạo mới.
- Cập nhật `total_spend`, `current_points`, `lifetime_points`, `current_tier_id`.
- Insert `loyalty_point_transactions`.
- Toàn bộ thao tác atomic và idempotent.

## 6. Claim/link flow

Offline loyalty flow:

```text
Brand staff nhập phone + customer_name + purchase_amount
-> hệ thống tìm/tạo brand_customer theo brand_id + phone_hash
-> tạo loyalty_account nếu chưa có
-> tính earned points theo loyalty_program
-> update total_spend/current_points/lifetime_points/current_tier_id
-> insert loyalty_point_transactions EARN
-> sinh claim token/QR nếu cần
```

Claim flow:

```text
User đăng nhập Closy bằng auth hiện tại
-> user nhập claim code hoặc quét QR
-> hệ thống hash token và tìm brand_customer_claims hợp lệ
-> kiểm tra token chưa hết hạn, chưa consumed
-> kiểm tra brand_customers.user_id IS NULL
-> set brand_customers.user_id = current_user.id
-> set brand_customers.claimed_at = now()
-> set brand_customer_claims.consumed_at = now()
-> update loyalty_accounts.user_id = current_user.id
-> từ đó user xem được điểm/hạng trong app
```

Không dùng OTP phone trong claim flow MVP.

## 7. Brand portal access và privacy

Brand không phải account login riêng.

```text
users = người thật đăng nhập
brands = tổ chức / nhãn hàng
brand_members = user nào có quyền quản trị brand nào
```

Brand Portal access phải check:

- `brands.status = ACTIVE`
- `brand_members.user_id = current_user.id`
- `brand_members.status = ACTIVE`
- role phù hợp: `OWNER`, `MANAGER`, `SUPPORT_STAFF`, `MARKETER`

Vì MVP không dùng `user_brand_consents`, privacy phải enforce bằng query/data visibility rule.

Brand được xem:

- brand customers thuộc brand đó
- loyalty account và transaction history của brand đó
- benefit redemption của brand đó
- chat giữa user và brand đó
- feedback/vote của user với brand item/sample của brand đó
- outfit interaction có chứa brand item của chính brand đó
- aggregate insight

Brand không được xem:

- raw wardrobe_items của user
- toàn bộ tủ đồ cá nhân của user
- outfit cá nhân không chứa brand item
- AI chat riêng của user
- dữ liệu app cá nhân không liên quan tới brand

## 8. Brand items và AI recommendation

Không dùng:

- `required_brand_item_id`
- `GenerateSampleTrialStyling`
- `GenerateDigitalSampleOutfit`
- sample trial usecase riêng

AI outfit recommendation chỉ mở rộng bằng flag:

```text
include_brand_items = true/false
```

Rule:

- Nếu `include_brand_items = false`, chỉ dùng wardrobe items của user.
- Nếu `include_brand_items = true`, `styling` gọi `brand/contract.ListEligibleBrandItemsForStyling(userID, filter)`.
- `styling` không tự query brand/loyalty tables.
- `brand` module tự quyết định brand item hợp lệ dựa trên brand active, item active, brand customer active/linked nếu cần, benefit/feature access nếu item là sample hoặc feature đặc biệt.
- AI tự chọn brand item phù hợp, không ép user truyền `required_brand_item_id`.

`outfit_items` target:

```text
outfit_id
fashion_item_id
item_context: USER_WARDROBE | BRAND_ITEM
```

## 9. Benefit và feature access

`brand_benefits` không chỉ là voucher. Nó có thể mở quyền truy cập feature.

Feature code MVP:

```text
SAMPLE_MIX_ACCESS
BRAND_ITEM_RECOMMENDATION
PRIORITY_BRAND_CHAT
```

Check quyền nằm trong `brand/contract`:

```text
CheckBrandFeatureAccess(userID, brandID, featureCode) -> bool
ListEligibleBrandItemsForStyling(userID, filter) -> []BrandItemStylingDTO
```

Không làm complex rule engine.

## 10. Phase plan cập nhật

```text
Phase 00: Alignment and repository audit
Phase 01: Giữ current auth/users schema, xác nhận không làm phone-first
Phase 02: Archive legacy community/resale
Phase 03: Fashion items migration
Phase 04: Outfit items context
Phase 05: Brand module
  05a Brand core: brands, brand_members, brand_customers nullable user_id
  05b Loyalty schema: brand_customer_id, brand_customer_claims, append-only ledger
  05c Unified loyalty points API
  05d Benefits and feature access
  05e Brand chat
  05f Privacy visibility
Phase 06: Brand items and sample feedback
Phase 07: Styling brand integration với include_brand_items
Phase 08: Seed demo and final validation
```

## 11. Out of scope MVP

Không làm trong MVP:

- phone-first identity
- tạo users UNVERIFIED từ brand offline purchase
- phone_otp_challenges
- user_brand_consents
- loyalty_point_lots
- remaining_points trong transaction
- brand_orders
- brand_order_items
- support_tickets
- return_exchange_requests
- campaign
- brand subscription / B2B billing
- required_brand_item_id
- sample trial usecase riêng
- complex rule engine
- multiple active loyalty programs per brand
- rolling yearly tier reset

