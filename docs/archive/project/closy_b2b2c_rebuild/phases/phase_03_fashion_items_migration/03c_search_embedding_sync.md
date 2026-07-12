# Phase 03c - Search, Embedding, and Sync Refactor

## Mục tiêu

Chuyển lexical search, vector search, AI context builder và Elasticsearch sync từ metadata trên `wardrobe_items` sang metadata trên `fashion_items`.

## Không làm trong phase này

```text
- Không đổi AI recommendation algorithm lớn.
- Không thêm brand item candidates.
- Không tạo new search engine.
```

## SQL lexical search

Nếu hiện có GIN/tsvector index trên `wardrobe_items`, tạo index tương đương trên `fashion_items`.

Ví dụ concept:

```sql
CREATE INDEX ... ON fashion_items USING gin(
  to_tsvector('simple',
    coalesce(color,'') || ' ' ||
    coalesce(style,'') || ' ' ||
    coalesce(material,'') || ' ' ||
    coalesce(pattern,'') || ' ' ||
    coalesce(fit,'') || ' ' ||
    coalesce(seasonality,'') || ' ' ||
    coalesce(description,'')
  )
);
```

Dùng config/language/index style hiện tại nếu repo đã có.

## Vector search

Move HNSW/vector index sang `fashion_items.embedding`.

Query wardrobe vector search phải join qua ownership:

```sql
SELECT wi.id AS wardrobe_item_id, fi.id AS fashion_item_id, ...
FROM wardrobe_items wi
JOIN fashion_items fi ON fi.id = wi.fashion_item_id
WHERE wi.user_id = $1
  AND wi.is_deleted = false
ORDER BY fi.embedding <=> $query_embedding
LIMIT $k;
```

Nếu current distance operator khác, giữ operator hiện có.

## AI context builder

Mọi context đưa vào LLM phải dùng metadata từ `fashion_items`.

DTO cho styling nên có cả:

```text
wardrobe_item_id
fashion_item_id
item_context = USER_WARDROBE
category/color/style/material/pattern/fit/seasonality/description
image_url
embedding optional
last_used_at from wardrobe_items
```

## Elasticsearch sync

Nếu có sync worker/document:

Document nên denormalize:

```text
wardrobe_item_id
fashion_item_id
user_id
metadata from fashion_items
ownership fields from wardrobe_items
```

Khi `fashion_items` metadata đổi, phải re-sync related wardrobe item docs.

Nếu hiện chưa có robust sync, ghi TODO nhưng không bỏ qua query SQL path.

## Tests

- Search by color/style/material vẫn trả item đúng.
- Vector search vẫn trả item của đúng user, không leak item user khác.
- AI context builder không còn đọc metadata cũ từ `wardrobe_items`.
- Elasticsearch document nếu có chứa `fashion_item_id` và metadata đúng.

## Acceptance checklist

- [ ] Lexical search đọc `fashion_items`.
- [ ] Vector search đọc `fashion_items.embedding`.
- [ ] Search vẫn filter theo `wardrobe_items.user_id`.
- [ ] Không leak item giữa users.
- [ ] AI context builder dùng DTO mới.
