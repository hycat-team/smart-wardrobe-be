# Phase 03b - Code Refactor to Read fashion_items

## Mục tiêu

Cập nhật entity/repository/usecase/DTO để app đọc metadata thời trang từ `fashion_items` thay vì trực tiếp từ `wardrobe_items`.

## Không làm trong phase này

```text
- Không drop metadata columns cũ khỏi wardrobe_items.
- Không đổi outfit_items.
- Không thêm brand_items.
```

## Entity/model changes

Tạo entity/model `FashionItem` trong module `wardrobe` hoặc shared domain hiện tại nếu repo đang dùng shared entities.

`WardrobeItem` sau refactor phải có:

```text
id
user_id
fashion_item_id
fashion_item embedded/loaded optional
purchase_price
status
item_type
last_used_at
is_deleted
created_at
updated_at
```

Không để API response thiếu metadata cũ. Handler phải join `fashion_items` và trả về response tương thích nếu FE hiện còn cần:

```text
category
image_url
color/style/material/pattern/fit/seasonality
description
```

## Repository changes

Các query list/get wardrobe item phải join:

```sql
wardrobe_items wi
JOIN fashion_items fi ON fi.id = wi.fashion_item_id
```

Các filter theo metadata phải đổi từ `wi.color` sang `fi.color`, tương tự:

```text
category_id -> fi.category_id
color -> fi.color
style -> fi.style
material -> fi.material
pattern -> fi.pattern
fit -> fi.fit
seasonality -> fi.seasonality
embedding -> fi.embedding
```

## Create wardrobe item flow

Khi user upload/create wardrobe item:

1. Tạo `fashion_items` trước với ảnh/metadata/processing state.
2. Tạo `wardrobe_items` trỏ `fashion_item_id`.
3. Nếu AI processing chạy async, update processing fields trên `fashion_items`, không update `wardrobe_items`.
4. Capacity chỉ tính `wardrobe_items`, không tính `fashion_items`.

## AI processing retry flow

Các retry fields chuyển sang `fashion_items`:

```text
processing_retry_count
processing_version
processing_started_at
last_processing_attempt_at
processing_error_reason
review_reason
```

Retry AI Vision phải lookup fashion item từ wardrobe item:

```text
wardrobe_item_id -> fashion_item_id -> fashion_items processing fields
```

Không xóa wardrobe item nếu AI processing fail.

## API compatibility

Nếu public API hiện trả `wardrobe_item.id`, tiếp tục trả `wardrobe_item.id` là user-facing ID.

Không expose `fashion_item_id` bắt buộc cho FE nếu chưa cần. Có thể expose internal/debug nếu API design hiện có pattern.

## Tests

Unit/usecase:

- Create wardrobe item tạo cả `fashion_items` và `wardrobe_items`.
- AI processing update `fashion_items`.
- Capacity check chỉ tính `wardrobe_items`.
- Get wardrobe item trả đủ metadata từ `fashion_items`.

Repository:

- List wardrobe items join đúng.
- Filter category/color/style đúng sau migration.
- Soft delete wardrobe item không xóa fashion item.

## Acceptance checklist

- [ ] Wardrobe APIs vẫn trả dữ liệu như trước.
- [ ] Metadata đọc từ `fashion_items`.
- [ ] New upload tạo `fashion_items` trước.
- [ ] AI retry/update dùng `fashion_items`.
- [ ] Không drop metadata cũ trước khi all tests pass.

## Lỗi cần tránh

- Trả `fashion_item_id` thay cho `wardrobe_item.id` trong API cũ làm FE vỡ.
- Tính capacity theo `fashion_items`.
- Update processing state trên `wardrobe_items` sau refactor.
