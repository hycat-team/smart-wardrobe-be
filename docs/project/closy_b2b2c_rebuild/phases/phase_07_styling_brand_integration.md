# Phase 07 - Styling Refactor and Brand Item Integration

## Mục tiêu

Tách/chuẩn hóa module `styling` cho AI outfit recommendation và mở rộng luồng recommendation hiện có để hỗ trợ brand items bằng flag `include_brand_items`.

## Không làm trong phase này

```text
- Không tạo GenerateSampleTrialStyling.
- Không tạo GenerateDigitalSampleOutfit.
- Không tạo required_brand_item_id.
- Không để styling query trực tiếp brand/loyalty tables.
- Không expose raw wardrobe cho brand.
- Không tạo new AI endpoint song song nếu endpoint cũ đã có.
```

## Endpoint target

Mở rộng endpoint hiện tại:

```text
POST /api/v1/ai/outfit-recommendations
```

Request body thêm:

```json
{
  "include_brand_items": true
}
```

Không thêm:

```json
{
  "required_brand_item_id": "..."
}
```

Nếu cần filter nhẹ sau này, có thể thêm `brand_id` hoặc `brand_item_type`, nhưng MVP chưa cần nếu không có yêu cầu FE.

## High-level flow

```text
1. User gọi AI outfit recommendation.
2. subscription.ReserveAIUsage nếu current flow có quota reservation.
3. styling gọi wardrobe.ListUserWardrobeItemsForStyling(userID, filter).
4. Nếu include_brand_items = true:
   - styling gọi brand.ListEligibleBrandItemsForStyling(userID, filter).
   - brand module tự check membership, benefit/feature access, brand/item status.
5. styling merge candidates:
   - user wardrobe items: item_context USER_WARDROBE
   - brand items: item_context BRAND_ITEM
6. Retrieval/rerank/prompt hiện có chọn outfit.
7. Save outfit + outfit_items:
   - outfit_items.fashion_item_id
   - outfit_items.item_context
8. FinalizeAIUsage nếu success.
9. RefundAIUsage nếu failure theo current quota rule.
```

## Candidate DTO

Unify internal styling candidate:

```text
fashion_item_id
item_context USER_WARDROBE|BRAND_ITEM
wardrobe_item_id nullable
brand_item_id nullable
brand_id nullable
brand_item_type nullable PRODUCT|SAMPLE
category
image_url
color/color_hex/hsl
style
material
pattern
fit
seasonality
description
last_used_at nullable
price nullable
source_label optional
```

`wardrobe_item_id` chỉ có với USER_WARDROBE.

`brand_item_id` chỉ có với BRAND_ITEM.

## Brand contract eligibility

Styling gọi:

```text
ListEligibleBrandItemsForStyling(userID, filter)
```

Brand module phải enforce:

```text
- user status ACTIVE
- brand status ACTIVE
- brand_customer status ACTIVE
- brand_item status ACTIVE
- PRODUCT: membership đủ cho recommendation MVP
- SAMPLE: cần SAMPLE_MIX_ACCESS hoặc sample public config nếu có
```

Styling không biết chi tiết loyalty/benefit.

## Recommendation behavior

Nếu `include_brand_items = false`:

```text
- Behavior giống hiện tại.
- Chỉ dùng wardrobe items của user.
```

Nếu `include_brand_items = true` nhưng user không eligible brand nào:

```text
- Không fail.
- Fallback về wardrobe-only recommendation.
- Response có thể kèm warning/metadata nếu API hiện có hỗ trợ.
```

Nếu brand candidates có nhưng AI không chọn:

```text
- Vẫn hợp lệ.
- include_brand_items không có nghĩa bắt buộc phải dùng brand item.
```

## Prompt/retrieval guidance

Không ép brand item vào outfit.

Prompt nên nói:

```text
You may include eligible brand items if they improve the outfit. Do not force them.
```

Rule reranker có thể bonus nhẹ cho brand item nếu include flag true, nhưng không làm outfit xấu đi.

MVP không cần thiết kế thuật toán hoàn toàn mới.

## Save outfit rules

Khi AI output chọn candidate:

```text
- USER_WARDROBE -> save item_context USER_WARDROBE.
- BRAND_ITEM -> save item_context BRAND_ITEM.
```

Validation:

```text
- USER_WARDROBE candidate must belong to user.
- BRAND_ITEM candidate must come from brand contract result of this request.
```

Không accept arbitrary brand fashion_item_id từ client.

## API response

Outfit response item nên có source info:

```json
{
  "fashion_item_id": "uuid",
  "item_context": "BRAND_ITEM",
  "brand_item": {
    "id": "uuid",
    "brand_id": "uuid",
    "brand_name": "Brand X",
    "item_type": "PRODUCT",
    "name": "White Shirt",
    "price": 199000
  }
}
```

For USER_WARDROBE:

```json
{
  "fashion_item_id": "uuid",
  "item_context": "USER_WARDROBE",
  "wardrobe_item_id": "uuid"
}
```

Keep backward compatibility if FE expects old shape; add fields without removing old fields when possible.

## Tests

- include_brand_items false -> no brand contract call, wardrobe-only behavior.
- include_brand_items true -> brand contract called.
- User not brand_customer -> no brand candidates, no fail.
- Eligible PRODUCT can appear as BRAND_ITEM.
- SAMPLE requires SAMPLE_MIX_ACCESS.
- AI output cannot inject brand item not returned by brand contract.
- Saved outfit_items have correct item_context.
- Quota refunded/finalized same as current success/failure rule.

## Acceptance checklist

- [ ] No `required_brand_item_id`.
- [ ] No separate sample generation usecase.
- [ ] Styling uses brand contract for eligibility.
- [ ] Brand item candidates are optional.
- [ ] outfit_items store correct item_context.
- [ ] No raw wardrobe exposed to brand.
