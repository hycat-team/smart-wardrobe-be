# Phase 03a - fashion_items Schema and Backfill

## Mục tiêu

Tách metadata thời trang khỏi `wardrobe_items` sang bảng lõi `fashion_items` để cùng dùng cho item cá nhân và item của brand.

## Không làm trong phase này

```text
- Không tạo brand_items.
- Không đổi outfit_items sang fashion_item_id trong phase này.
- Không drop metadata columns khỏi wardrobe_items ngay.
- Không tạo garment_specs.
- Không tạo module garment.
```

## Current source fields

Từ `wardrobe_items` hiện tại, các field sau chuyển sang `fashion_items`:

```text
category_id
image_url
image_public_id
color
color_hex
color_hue
color_saturation
color_lightness
style
material
pattern
fit
seasonality
description
embedding
processing_retry_count
processing_version
processing_started_at
last_processing_attempt_at
processing_error_reason
review_reason
created_at
updated_at
```

Các field ở lại `wardrobe_items`:

```text
id
user_id
status
item_type
last_used_at
is_deleted
created_at
updated_at
```

`price` nếu còn dùng thì đổi ý nghĩa thành `purchase_price` trên `wardrobe_items`.

## Schema target

Tạo bảng `fashion_items`:

```text
id UUID PK
category_id UUID FK categories(id)
image_url VARCHAR(500)
image_public_id VARCHAR(255)
color VARCHAR(50)
color_hex VARCHAR(7)
color_hue DOUBLE PRECISION
color_saturation DOUBLE PRECISION
color_lightness DOUBLE PRECISION
style VARCHAR(100)
material VARCHAR(100)
pattern VARCHAR(100)
fit VARCHAR(50)
seasonality VARCHAR(100)
description TEXT
embedding vector(768) hoặc dimension hiện tại
processing_retry_count INT
processing_version INT
processing_started_at TIMESTAMP NULL
last_processing_attempt_at TIMESTAMP NULL
processing_error_reason TEXT NULL
review_reason VARCHAR(100) NULL
created_at TIMESTAMP
updated_at TIMESTAMP
```

Dimension vector phải lấy từ schema hiện tại. Nếu hiện không phải 768, dùng dimension hiện tại.

## Backfill strategy bắt buộc

Để migration đơn giản và giữ outfit_items cũ dễ migrate về sau:

```text
fashion_items.id = old wardrobe_items.id
wardrobe_items.fashion_item_id = old wardrobe_items.id
```

Các bước migration:

1. Create `fashion_items`.
2. Insert data từ `wardrobe_items` sang `fashion_items`, dùng cùng `id`.
3. Add column `wardrobe_items.fashion_item_id UUID NULL`.
4. Backfill `wardrobe_items.fashion_item_id = wardrobe_items.id`.
5. Add FK `wardrobe_items.fashion_item_id -> fashion_items.id`.
6. Để `fashion_item_id` nullable hoặc NOT NULL tùy trạng thái data, nhưng sau backfill nên NOT NULL nếu toàn bộ rows có dữ liệu.
7. Rename `price` -> `purchase_price` nếu field `price` tồn tại và còn cần giữ.

## Index rules

Không tạo global unique index trên `wardrobe_items.fashion_item_id`.

Lý do: sau này nhiều user có thể sở hữu cùng một brand product/fashion item. Nếu cần chống duplicate trong tủ một user, chỉ cân nhắc unique partial:

```sql
CREATE UNIQUE INDEX ... ON wardrobe_items(user_id, fashion_item_id)
WHERE is_deleted = false;
```

Không tự thêm unique này nếu current business cho phép user có duplicate item.

Tạo index cho join phổ biến:

```sql
CREATE INDEX ... ON wardrobe_items(user_id, fashion_item_id);
CREATE INDEX ... ON fashion_items(category_id);
```

Vector index và lexical index xử lý ở Phase 03c.

## Data validation queries

Sau migration, chạy kiểm tra:

```sql
SELECT COUNT(*) FROM wardrobe_items WHERE fashion_item_id IS NULL;

SELECT COUNT(*)
FROM wardrobe_items wi
LEFT JOIN fashion_items fi ON fi.id = wi.fashion_item_id
WHERE fi.id IS NULL;

SELECT COUNT(*) FROM fashion_items;
SELECT COUNT(*) FROM wardrobe_items;
```

Kỳ vọng:

```text
- Không có wardrobe_items thiếu fashion_item_id.
- Không có broken FK.
- Số fashion_items backfill bằng số wardrobe_items legacy tại thời điểm migration.
```

## Rollback note

Nếu migration framework yêu cầu down migration:

```text
- Drop FK wardrobe_items.fashion_item_id
- Drop column wardrobe_items.fashion_item_id
- Drop fashion_items
- Rename purchase_price về price nếu đã rename
```

Không rollback data nếu production đã chạy phase sau.

## Acceptance checklist

- [ ] `fashion_items` được tạo.
- [ ] Legacy wardrobe items được backfill sang fashion_items cùng ID.
- [ ] `wardrobe_items.fashion_item_id` được backfill.
- [ ] Không có global unique constraint sai trên `wardrobe_items.fashion_item_id`.
- [ ] Metadata columns cũ chưa bị drop trước khi code refactor xong.
