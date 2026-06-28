# Phase 04 - outfit_items uses fashion_item_id and item_context

## Mục tiêu

Cập nhật `outfit_items` để một outfit có thể chứa cả đồ cá nhân của user và item từ brand. Bảng `outfit_items` phải trỏ về `fashion_items`, đồng thời lưu `item_context` để biết item này được dùng trong outfit dưới vai trò nào.

## Không làm trong phase này

```text
- Không tạo brand_items.
- Không tích hợp AI brand recommendation.
- Không tạo sample_outfit_trials.
- Không tạo sample_trial_items.
- Không tạo required_brand_item_id.
```

## Schema target

Hiện tại `outfit_items.item_id` đang trỏ `wardrobe_items.id`.

Target:

```text
outfit_id UUID FK outfits(id)
fashion_item_id UUID FK fashion_items(id)
item_context VARCHAR(50) NOT NULL
position_x DOUBLE PRECISION
position_y DOUBLE PRECISION
scale DOUBLE PRECISION
layer_order SMALLINT
created_at
updated_at
```

`item_context` values:

```text
USER_WARDROBE
BRAND_ITEM
```

Composite key target có thể là:

```text
(outfit_id, fashion_item_id, item_context)
```

Nếu muốn giữ gần schema cũ hơn, có thể dùng `(outfit_id, fashion_item_id)` nhưng phải xác nhận không có case cùng một `fashion_item_id` xuất hiện 2 lần trong outfit với 2 context khác nhau.

Khuyến nghị dùng `(outfit_id, fashion_item_id, item_context)` để tránh ambiguity.

## Backfill strategy

Vì Phase 03 đã set:

```text
wardrobe_items.fashion_item_id = old wardrobe_items.id
fashion_items.id = old wardrobe_items.id
```

Có thể migrate:

```sql
ALTER TABLE outfit_items ADD COLUMN fashion_item_id UUID NULL;
ALTER TABLE outfit_items ADD COLUMN item_context VARCHAR(50) NULL;

UPDATE outfit_items oi
SET fashion_item_id = wi.fashion_item_id,
    item_context = 'USER_WARDROBE'
FROM wardrobe_items wi
WHERE wi.id = oi.item_id;
```

Sau validation:

```text
- fashion_item_id NOT NULL
- item_context NOT NULL
- FK fashion_item_id -> fashion_items(id)
- drop old FK item_id -> wardrobe_items(id)
- drop/rename old item_id after code switch
```

Nếu migration framework không thích rename/drop cùng lúc, tách thành 2 migrations.

## Validation queries

```sql
SELECT COUNT(*) FROM outfit_items WHERE fashion_item_id IS NULL;
SELECT COUNT(*) FROM outfit_items WHERE item_context IS NULL;

SELECT COUNT(*)
FROM outfit_items oi
LEFT JOIN fashion_items fi ON fi.id = oi.fashion_item_id
WHERE fi.id IS NULL;
```

Kỳ vọng tất cả = 0.

## Code refactor

### DTO/usecase save outfit

Khi lưu outfit, input item nên có:

```text
fashion_item_id
item_context
position_x
position_y
scale
layer_order
```

Trong user-created outfit hiện tại, FE có thể vẫn gửi `wardrobe_item_id`. Backend có thể translate:

```text
wardrobe_item_id -> wardrobe_items.fashion_item_id
item_context = USER_WARDROBE
```

Không bắt FE đổi toàn bộ ngay nếu chưa cần.

### Validation rule

Nếu `item_context = USER_WARDROBE`:

```text
- fashion_item_id phải thuộc một wardrobe_item của user
- wardrobe_item không bị deleted
```

Nếu `item_context = BRAND_ITEM`:

```text
- fashion_item_id phải thuộc brand_items ACTIVE
- eligibility sẽ enforce ở Phase 07 khi AI brand integration hoàn chỉnh
```

Trong phase này brand_items chưa tồn tại, nên chỉ cần prepare validation branch nhưng chưa bật BRAND_ITEM save từ public API nếu chưa có source.

## Outfit display

Khi render outfit item:

```text
- load fashion_items metadata
- nếu item_context USER_WARDROBE, join wardrobe_items để lấy ownership fields nếu cần
- nếu item_context BRAND_ITEM, phase sau join brand_items để lấy brand/name/price
```

## Tests

Migration tests:

- Existing outfit_items backfill thành `fashion_item_id` đúng.
- Existing outfit_items đều có `item_context = USER_WARDROBE`.

Usecase tests:

- Save outfit với wardrobe item cũ translate sang fashion_item_id.
- User không thể dùng fashion_item không thuộc wardrobe khi context USER_WARDROBE.
- Outfit response vẫn hiển thị metadata đúng.

## Acceptance checklist

- [ ] Existing outfits không mất item.
- [ ] `outfit_items` trỏ `fashion_items`.
- [ ] Có `item_context`.
- [ ] Existing items backfill USER_WARDROBE.
- [ ] Không có sample trial tables.
- [ ] API cũ vẫn tương thích hoặc có migration FE rõ ràng.

## Lỗi cần tránh

- Chỉ lưu `fashion_item_id` mà quên `item_context`.
- Cho user save BRAND_ITEM khi chưa có eligibility check.
- Làm mất position/layer_order cũ.
