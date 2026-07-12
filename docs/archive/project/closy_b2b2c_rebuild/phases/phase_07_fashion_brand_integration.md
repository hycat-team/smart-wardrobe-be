# Phase 07 - Fashion Refactor and Brand Item Integration

## Mục tiêu

Chuẩn hóa module `fashion` quản lý `FashionItem`, dịch vụ AI outfit recommendation, và **AI Chat cá nhân (Personal AI Chatbot)**. Mở rộng luồng recommendation hiện có để hỗ trợ trộn sản phẩm của brand thông qua flag `include_brand_items`.

> [!IMPORTANT]
> **Fashion Module làm trung tâm cho toàn bộ các chức năng AI và Fashion Items**:
> 1. Tầng nghiệp vụ lưu trữ, truy xuất và cập nhật `FashionItem` nằm hoàn toàn dưới sự quản lý của module `fashion`.
> 2. Luồng lưu outfit chạm vào `FashionItem` (như trong `wardrobe` module) sẽ gọi thông qua Service Contract của `fashion` thay vì trực tiếp truy cập repository của `FashionItem` như trước đây.
> 3. Toàn bộ các chức năng AI bao gồm: AI Phối đồ (Recommendations), AI Vision (Phân tích ảnh trích xuất metadata), và AI Chat cá nhân (tư vấn trang phục dựa trên tủ đồ của user) đều được quy tụ về module `fashion`.

## Không làm trong phase này

```text
- Không tạo GenerateSampleTrialStyling.
- Không tạo GenerateDigitalSampleOutfit.
- Không tạo required_brand_item_id.
- Không để fashion query trực tiếp brand/loyalty tables.
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
1. User gọi AI outfit recommendation hoặc AI Chat.
2. subscription.ReserveAIUsage nếu current flow có quota reservation.
3. fashion gọi wardrobe.ListUserWardrobeItemsForStyling(userID, filter) để lấy danh sách đồ cá nhân của user.
4. Nếu include_brand_items = true:
   - fashion gọi brand.ListEligibleBrandItemsForStyling(userID, filter).
   - brand module tự check membership, benefit/feature access, brand/item status.
5. fashion merge candidates:
   - user wardrobe items: item_context USER_WARDROBE
   - brand items: item_context BRAND_ITEM (đối chiếu fashion_item metadata trực tiếp trong database của fashion)
6. Retrieval/rerank/prompt hiện có chọn outfit hoặc trả lời tin nhắn chat.
7. Save outfit + outfit_items:
   - Gọi fashion contract để xác thực fashion_item_id và lưu outfit_items.
   - Lưu outfit_items.item_context tương ứng.
8. FinalizeAIUsage nếu success.
9. RefundAIUsage nếu failure theo current quota rule.
```

## Candidate DTO

Unify internal fashion candidate:

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

Fashion gọi:

```text
ListEligibleBrandItemsForStyling(userID, filter)
```

Brand module phải enforce:
- user status ACTIVE
- brand status ACTIVE
- brand_customer status ACTIVE
- brand_item status ACTIVE
- PRODUCT: membership đủ cho recommendation MVP
- SAMPLE: cần SAMPLE_MIX_ACCESS hoặc sample public config nếu có

Fashion không biết chi tiết loyalty/benefit.

## Recommendation behavior

Nếu `include_brand_items = false`:
- Behavior giống hiện tại.
- Chỉ dùng wardrobe items của user.

Nếu `include_brand_items = true` nhưng user không eligible brand nào:
- Không fail.
- Fallback về wardrobe-only recommendation.
- Response có thể kèm warning/metadata nếu API hiện có hỗ trợ.

Nếu brand candidates có nhưng AI không chọn:
- Vẫn hợp lệ.
- include_brand_items không có nghĩa bắt buộc phải dùng brand item.

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
- USER_WARDROBE -> save item_context USER_WARDROBE.
- BRAND_ITEM -> save item_context BRAND_ITEM.

Validation:
- USER_WARDROBE candidate must belong to user.
- BRAND_ITEM candidate must come from brand contract result of this request.

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
- [ ] Fashion uses brand contract for eligibility.
- [ ] Brand item candidates are optional.
- [ ] outfit_items store correct item_context.
- [ ] No raw wardrobe exposed to brand.
