# Phase 05f - Brand Privacy and Data Visibility

## Mục tiêu

Vì MVP không dùng `user_brand_consents`, phải enforce privacy bằng rule query/data visibility rõ ràng. Brand không được xem raw wardrobe cá nhân của user.

## Không làm trong phase này

```text
- Không tạo user_brand_consents.
- Không tạo opt-in dashboard phức tạp.
- Không expose wardrobe_items raw cho brand.
- Không expose AI chat cá nhân cho brand.
```

## Data brand được xem

Brand Portal được xem dữ liệu thuộc brand đó:

```text
- brand profile của chính brand
- brand members của chính brand
- brand customers của chính brand
- loyalty account của customer thuộc brand
- loyalty point transactions của customer thuộc brand
- benefit redemptions thuộc brand
- brand conversations/messages thuộc brand
- brand_items thuộc brand
- digital_sample_responses thuộc brand_items của brand
- outfit interaction chỉ khi outfit_items có item_context BRAND_ITEM và brand_item thuộc brand
- aggregate insight không lộ raw wardrobe cá nhân
```

## Data brand không được xem

```text
- toàn bộ wardrobe_items của user
- fashion_items cá nhân của user nếu không liên quan brand item
- outfit cá nhân không chứa brand item của brand
- AI chat riêng của user
- subscription/payment riêng của user ngoài loyalty brand
- raw personal metadata như màu/style/chất liệu của toàn bộ tủ đồ
```

## Query rule bắt buộc

Mọi Brand Portal query phải có `brand_id` filter từ path/auth context.

Không nhận `brand_id` tự do trong body nếu path đã có brandId.

Không query theo user_id đơn thuần trong Brand Portal nếu thiếu brand filter.

Ví dụ đúng:

```sql
SELECT ...
FROM loyalty_accounts
WHERE brand_id = $brandID AND brand_customer_id = $brandCustomerID;
```

Nếu query theo linked user thì vẫn phải đi qua brand scope:

```sql
SELECT ...
FROM loyalty_accounts
WHERE brand_id = $brandID AND user_id = $userID;
```

Ví dụ sai:

```sql
SELECT ...
FROM wardrobe_items
WHERE user_id = $userID;
```

## Aggregate insight MVP

Nếu cần dashboard demo, chỉ dùng aggregate từ brand-related interactions:

```text
- number of responses per sample
- LIKE / WOULD_BUY count per sample
- average rating per sample
- count of outfits containing brand_items
- top brand item categories/colors based on brand_items, not user raw wardrobe
```

Không làm wardrobe insight cá nhân hóa trong MVP.

## Tests

- Staff Brand A không xem được customer/transaction/chat Brand B.
- Brand Portal API không trả wardrobe_items raw.
- Sample responses chỉ trả responses cho brand_items thuộc brand.
- Outfit interaction query chỉ trả outfit có brand item thuộc brand.

## Acceptance checklist

- [ ] Không có user_brand_consents table.
- [ ] Privacy rule được document trong code/service comments.
- [ ] Brand queries luôn filter brand_id.
- [ ] Không expose raw wardrobe.
- [ ] Không expose private AI chat.
