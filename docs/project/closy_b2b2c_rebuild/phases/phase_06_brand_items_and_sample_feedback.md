# Phase 06 - Brand Items and Sample Feedback

## Mục tiêu

Thêm `brand_items` để brand quản lý sản phẩm thật và sample thử nghiệm, đồng thời thêm feedback/vote cho sample. Brand items dùng chung `fashion_items` làm item core.

## Không làm trong phase này

```text
- Không tạo digital_sample_variants.
- Không tạo sample_outfit_trials.
- Không tạo sample_trial_items.
- Không tạo brand_orders.
- Không tự động thêm brand product vào wardrobe khi mua vì chưa có order flow.
- Không tích hợp AI recommendation trong phase này; phase 07 làm.
```

## Schema target

### brand_items

```text
id UUID PK
brand_id UUID FK brands(id)
fashion_item_id UUID FK fashion_items(id)
product_code VARCHAR(100) NULL
name VARCHAR(255)
description TEXT NULL
price DECIMAL(12,2) NULL
item_type VARCHAR(50)
status VARCHAR(50)
created_at
updated_at
```

Item type:

```text
PRODUCT
SAMPLE
```

Status:

```text
DRAFT
ACTIVE
ARCHIVED
```

Recommended constraints:

```text
unique(fashion_item_id) if one fashion item can only be wrapped by one brand item.
unique(brand_id, product_code) where product_code is not null.
```

Do not create global unique on product_code alone.

### digital_sample_responses

```text
id UUID PK
brand_item_id UUID FK brand_items(id)
user_id UUID FK users(id)
outfit_id UUID FK outfits(id) NULL
vote_type VARCHAR(50) NULL
rating INT NULL
feedback_text TEXT NULL
created_at TIMESTAMP
```

Vote type:

```text
LIKE
DISLIKE
WOULD_BUY
NOT_INTERESTED
```

Rule:

```text
- brand_item_id must reference brand_items.item_type = SAMPLE.
- User must be ACTIVE.
- Brand item must be ACTIVE.
- Optional: unique(user_id, brand_item_id) if only one response per user/sample.
```

Postgres cannot enforce conditional FK on item_type directly. Enforce in usecase/repository.

## Brand Portal APIs

```text
GET /api/v1/brand-portal/brands/:brandId/brand-items
POST /api/v1/brand-portal/brands/:brandId/brand-items
PATCH /api/v1/brand-portal/brand-items/:itemId
```

Permissions:

```text
OWNER/MANAGER/MARKETER can manage brand_items.
SUPPORT_STAFF cannot manage brand_items by default.
```

Create brand item flow:

```text
1. Check brand permission.
2. Validate item_type PRODUCT/SAMPLE.
3. Create fashion_item with image/metadata.
4. Create brand_item wrapping fashion_item.
5. If AI metadata processing is needed, schedule same AI processing mechanism on fashion_item.
```

Important:

```text
- BRAND_ITEM does not create wardrobe_item.
- SAMPLE never counts into user wardrobe capacity.
- PRODUCT also does not count into user wardrobe capacity unless a future purchase/order/import flow explicitly creates wardrobe_item.
```

## User-facing sample response API

```text
POST /api/v1/brand-items/:itemId/responses
```

Body:

```json
{
  "vote_type": "LIKE",
  "rating": 4,
  "feedback_text": "Looks good",
  "outfit_id": "nullable uuid"
}
```

Rules:

```text
- User must be ACTIVE.
- itemId is brand_items.id, not fashion_items.id.
- brand_items.item_type must be SAMPLE.
- brand_items.status must be ACTIVE.
- If outfit_id provided, outfit must belong to user and contain this brand item's fashion_item_id with item_context BRAND_ITEM.
```

## Brand item DTO for styling

For Phase 07, prepare DTO:

```text
brand_item_id
brand_id
fashion_item_id
item_type PRODUCT|SAMPLE
name
description
price
metadata from fashion_items
feature requirements if needed
```

## Tests

- Brand staff creates PRODUCT brand item -> fashion_item + brand_item created.
- Brand staff creates SAMPLE -> fashion_item + brand_item created.
- SAMPLE does not create wardrobe_item.
- User cannot response to PRODUCT via sample response API.
- User response to SAMPLE saved.
- User cannot attach outfit_id not owned by them.
- Brand A cannot update Brand B item.

## Acceptance checklist

- [ ] brand_items table exists.
- [ ] digital_sample_responses table exists.
- [ ] Brand item create wraps fashion_item.
- [ ] Sample feedback works only for SAMPLE.
- [ ] No sample trial tables.
- [ ] No brand order flow.
