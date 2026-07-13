# Closy Fashion Items Migration Guide

Tài liệu này chốt lại hướng migration schema MVP cho Closy B2B2C theo hướng **tối giản nhưng không duplicate metadata**.

Đây là hướng mới thay cho các phương án trước như:

```text
garment_specs
sample_outfit_trials
sample_trial_items
digital_sample_variants
DIGITAL_SAMPLE_VARIANT
```

Các phương án trên không dùng trong MVP nếu không có lý do mới thật sự mạnh.

---

# Mục tiêu

Mục tiêu của migration này:

- Dùng một bảng item lõi chung cho cả item của user và sample của brand.
- Không duplicate metadata thời trang giữa `wardrobe_items` và `digital_samples`.
- Không biến digital sample thành item nằm trong tủ đồ user.
- Cho phép digital sample xuất hiện trong outfit của user như một item được phối.
- Giữ Digital Sample Lab gắn trực tiếp với AI Styling.
- Giữ schema đủ đơn giản để tránh over-engineering.

---

# Nguyên tắc quan trọng cho agent

## Schema hiện tại là nguồn gốc

Agent bắt buộc phải dựa vào schema hiện tại trong repo / migration / database dump làm nguồn chính.

Các field trong tài liệu này là **schema định hướng**, không phải DDL cuối cùng để copy y nguyên.

Trước khi đề xuất migration, agent phải kiểm tra:

```text
- schema SQL hiện tại
- migration hiện tại
- shared entities hiện tại
- repository hiện tại
- usecase hiện tại
- handler/DTO hiện tại
- search / embedding / Elasticsearch sync hiện tại
- Swagger annotations hiện tại
```

Nếu field trong tài liệu này khác với schema hiện tại, agent phải:

```text
- preserve field hiện có nếu nó vẫn phục vụ core logic
- giải thích rõ nếu đề xuất bỏ field
- giải thích rõ nếu đề xuất đổi tên field
- không tự ý xoá field chỉ vì tài liệu này không liệt kê
```

## Không code ngay

Agent không được code/migration ngay ở bước đầu.

Agent phải cập nhật lại report phân tích trước, gồm:

```text
- schema hiện tại
- schema mục tiêu
- mapping field
- migration plan
- rủi ro
- task list
```

Chỉ sau khi được duyệt mới tạo migration/code.

---

# Quyết định kiến trúc chốt

## Dùng bảng `fashion_items` làm item lõi

Thay vì tách `garment_specs`, ta dùng:

```text
fashion_items
```

`fashion_items` là bảng lõi chứa:

```text
metadata thời trang
ảnh item
embedding
trạng thái xử lý AI
```

Bảng này không trả lời câu hỏi item thuộc ai.

Bảng này chỉ trả lời:

```text
Item này trông như thế nào?
Nó thuộc category nào?
Màu gì?
Style gì?
Material gì?
Embedding gì?
Ảnh nào?
AI đã xử lý ra sao?
```

## `wardrobe_items` chỉ còn là wrapper cho item user sở hữu

`wardrobe_items` vẫn giữ tên bảng hiện tại để giảm phá vỡ code.

Nhưng sau migration, `wardrobe_items` không còn chứa metadata thời trang chính nữa.

Nó chỉ chứa:

```text
user_id
fashion_item_id
purchase/status/lifecycle của item trong tủ user
```

`wardrobe_items` trả lời câu hỏi:

```text
User nào sở hữu item này?
Item này có nằm trong tủ cá nhân không?
Item này có tính vào wardrobe capacity không?
Item này lần cuối được dùng khi nào?
```

## `digital_samples` là wrapper cho sample của brand

`digital_samples` trỏ tới `fashion_items`.

Nó chứa thông tin riêng của sample:

```text
brand_id
name
description hiển thị
target_price
status
```

`digital_samples` trả lời câu hỏi:

```text
Brand nào đang test sample này?
Sample này đang draft/active/archive?
Tên display của sample là gì?
Giá dự kiến là bao nhiêu?
```

## `outfit_items` trỏ thẳng tới `fashion_items`

Đây là điểm giúp schema đơn giản.

Một outfit có thể chứa:

```text
fashion_item từ wardrobe_items của user
fashion_item từ digital_samples của brand
```

Vì vậy `outfit_items` không cần:

```text
item_source_type
wardrobe_item_id
digital_sample_id
sample_trial_items
```

Chỉ cần:

```text
fashion_item_id
```

Muốn biết item đó là của user hay sample brand thì query qua wrapper table:

```text
fashion_item_id nằm trong wardrobe_items -> item user sở hữu
fashion_item_id nằm trong digital_samples -> sample brand
```

---

# Module mục tiêu

Giữ module tinh gọn:

```text
internal/modules/
├── identity/
├── subscription/
├── wardrobe/
├── styling/
└── brand/
```

Trong đó:

| Module | Trách nhiệm |
|---|---|
| `identity` | Auth, user, refresh token |
| `subscription` | Premium, quota, payment, AI usage entitlement |
| `wardrobe` | `fashion_items`, `wardrobe_items`, `digital_samples`, `outfits`, `outfit_items`, sample response |
| `styling` | AI outfit recommendation, AI chat, digital sample outfit generation |
| `brand` | Brand CRM, members, customers, loyalty, campaign, benefit, support |

Không tạo module riêng:

```text
garment
samplelab
```

trong MVP.

---

# Schema mục tiêu theo hướng tối giản

## Bảng `fashion_items`

Vai trò:

```text
Item lõi dùng chung cho user wardrobe item và brand digital sample.
```

Field tham khảo:

```text
fashion_items
- id
- category_id
- image_url
- image_public_id
- color
- color_hex
- color_hue
- color_saturation
- color_lightness
- style
- material
- pattern
- fit
- seasonality
- description
- embedding
- processing_retry_count
- processing_version
- processing_started_at
- last_processing_attempt_at
- processing_error_reason
- review_reason
- created_at
- updated_at
```

Nguồn field tham khảo từ schema hiện tại của `wardrobe_items`:

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

Ghi chú:

- Nếu schema hiện tại có field khác phục vụ AI processing/search, agent phải preserve hoặc giải thích nếu bỏ.
- `embedding` chỉ nên chuyển sang `fashion_items` nếu embedding hiện tại đại diện cho metadata/style/image của item lõi.
- Nếu embedding hiện tại phụ thuộc hoàn toàn vào ownership của user, agent phải báo lại trước khi migration.
- `category_id` vẫn trỏ tới `categories`.

---

## Bảng `wardrobe_items`

Vai trò:

```text
Item thuộc tủ đồ cá nhân của user.
```

Field tham khảo sau migration:

```text
wardrobe_items
- id
- user_id
- fashion_item_id
- purchase_price nullable
- status
- item_type
- last_used_at
- is_deleted
- created_at
- updated_at
```

Mapping từ schema hiện tại:

| Field hiện tại | Hướng xử lý |
|---|---|
| `id` | Giữ ở `wardrobe_items.id` |
| `user_id` | Giữ |
| `category_id` | Chuyển sang `fashion_items.category_id` |
| `image_url` | Chuyển sang `fashion_items.image_url` |
| `image_public_id` | Chuyển sang `fashion_items.image_public_id` |
| `color` | Chuyển sang `fashion_items.color` |
| `color_hex` | Chuyển sang `fashion_items.color_hex` |
| `color_hue` | Chuyển sang `fashion_items.color_hue` |
| `color_saturation` | Chuyển sang `fashion_items.color_saturation` |
| `color_lightness` | Chuyển sang `fashion_items.color_lightness` |
| `style` | Chuyển sang `fashion_items.style` |
| `material` | Chuyển sang `fashion_items.material` |
| `pattern` | Chuyển sang `fashion_items.pattern` |
| `fit` | Chuyển sang `fashion_items.fit` |
| `seasonality` | Chuyển sang `fashion_items.seasonality` |
| `description` | Chuyển sang `fashion_items.description` |
| `embedding` | Chuyển sang `fashion_items.embedding`, nếu đúng loại embedding item-level |
| `price` | Đổi nghĩa thành `purchase_price` nếu còn cần |
| `status` | Giữ ở `wardrobe_items.status` |
| `item_type` | Giữ ở `wardrobe_items.item_type` |
| `last_used_at` | Giữ |
| `processing_*` | Chuyển sang `fashion_items`, vì đây là trạng thái xử lý item lõi |
| `processing_error_reason` | Chuyển sang `fashion_items` |
| `review_reason` | Chuyển sang `fashion_items` |
| `is_deleted` | Giữ ở `wardrobe_items.is_deleted` |
| `created_at` | Giữ ở `wardrobe_items`, đồng thời có thể copy sang `fashion_items` cho data cũ |
| `updated_at` | Giữ ở `wardrobe_items`, đồng thời có thể copy sang `fashion_items` cho data cũ |

Ghi chú:

- Chỉ `wardrobe_items` mới tính vào wardrobe capacity.
- Digital sample không được lưu vào `wardrobe_items`.
- Nếu muốn giữ backward compatibility API, có thể giữ `wardrobe_items.id` là ID user-facing của wardrobe item.
- `fashion_item_id` có thể được backfill bằng chính old `wardrobe_items.id` để migration dễ hơn.

---

## Bảng `digital_samples`

Vai trò:

```text
Sample item của brand dùng để test bằng AI Styling với tủ đồ user.
```

Field tham khảo:

```text
digital_samples
- id
- brand_id
- fashion_item_id
- name
- description nullable
- target_price nullable
- status
- created_at
- updated_at
```

Status tối giản:

```text
DRAFT
ACTIVE
ARCHIVED
```

Ghi chú:

- `digital_samples` không chứa metadata màu/style/material chính.
- Metadata đọc qua `digital_samples.fashion_item_id -> fashion_items.id`.
- `description` ở `digital_samples` là mô tả hiển thị/concept của brand, không phải metadata AI bắt buộc.
- Nếu không cần display description riêng, có thể dùng `fashion_items.description`.

Không dùng ở MVP:

```text
digital_sample_variants
```

Nếu brand muốn test nhiều màu/mẫu, tạo nhiều dòng `digital_samples` riêng.

---

## Bảng `outfits`

Giữ gần schema hiện tại.

Field tham khảo:

```text
outfits
- id
- user_id
- name
- description
- cover_image_url
- cover_public_id
- outfit_source
- status
- is_deleted
- created_at
- updated_at
```

Field mới đề xuất:

```text
outfit_source
```

Giá trị:

```text
USER_CREATED
AI_RECOMMENDATION
DIGITAL_SAMPLE_LAB
```

Ghi chú:

- `outfit_source` giúp biết outfit được tạo từ user, AI recommendation hay Digital Sample Lab.
- Nếu chưa muốn sửa nhiều, có thể thêm sau. Nhưng khuyến nghị thêm trong MVP vì giúp brand insight rất nhiều.

---

## Bảng `outfit_items`

Hiện tại bảng này trỏ tới `wardrobe_items`.

Sau migration, bảng này nên trỏ tới `fashion_items`.

Field tham khảo:

```text
outfit_items
- outfit_id
- fashion_item_id
- position_x
- position_y
- scale
- layer_order
- created_at
- updated_at
```

Ghi chú:

- Có thể giữ composite primary key tương tự hiện tại: `(outfit_id, fashion_item_id)`.
- Không cần `item_source_type`.
- Không cần `wardrobe_item_id`.
- Không cần `digital_sample_id`.
- Không cần `sample_trial_items`.

Một outfit từ Digital Sample Lab sẽ có:

```text
outfits.outfit_source = DIGITAL_SAMPLE_LAB
```

và `outfit_items` gồm:

```text
fashion_item_id của item user sở hữu
fashion_item_id của digital sample brand
```

Domain rule:

```text
- Với outfit thường hoặc AI recommendation B2C, fashion_item_id trong outfit_items phải thuộc wardrobe_items của user.
- Với outfit_source = DIGITAL_SAMPLE_LAB, outfit có thể chứa fashion_item_id thuộc wardrobe_items của user và fashion_item_id thuộc digital_samples đang active.
```

Rule này nên enforce ở usecase, không cần over-engineer DB constraint.

---

## Bảng `digital_sample_responses`

Vai trò:

```text
Lưu vote/rating/feedback của user đối với digital sample.
```

Field tham khảo:

```text
digital_sample_responses
- id
- digital_sample_id
- user_id
- outfit_id nullable
- vote_type nullable
- rating nullable
- feedback_text nullable
- created_at
```

`vote_type` tham khảo:

```text
LIKE
DISLIKE
WOULD_BUY
NOT_INTERESTED
```

Ghi chú:

- Gộp vote và feedback vào một bảng để tránh over-engineering.
- `outfit_id` nullable để support hai trường hợp:
  - User feedback sau khi phối sample vào outfit.
  - User vote sample trực tiếp chưa phối.
- Không cần `outfit_item_id` ở MVP vì `digital_sample_id + outfit_id` đã đủ để biết feedback gắn với sample nào trong outfit nào.

---

# Luồng migration dữ liệu hiện có

## Ý tưởng chính

Dữ liệu hiện tại nằm trong `wardrobe_items`.

Migration sẽ split dữ liệu này thành:

```text
fashion_items      # metadata/asset/AI processing
wardrobe_items     # ownership/lifecycle của user
```

Đối với dữ liệu cũ, có thể dùng:

```text
fashion_items.id = old wardrobe_items.id
wardrobe_items.fashion_item_id = old wardrobe_items.id
```

Cách này giúp:

- dữ liệu cũ dễ backfill
- outfit_items hiện tại có thể chuyển từ `item_id` sang `fashion_item_id` mà không đổi giá trị
- giảm rủi ro mất liên kết outfit cũ

---

# Kế hoạch migration đề xuất

## Phase 0: Report trước, chưa code

Agent cần cập nhật report trước.

Report phải có:

```text
- kiểm kê schema hiện tại
- mapping field từ wardrobe_items sang fashion_items/wardrobe_items
- tác động tới outfit_items
- tác động tới repository/usecase/handler
- tác động tới AI styling/search/embedding
- tác động tới API response
- kế hoạch migration dữ liệu
- rollback strategy nếu cần
```

## Phase 1: Tạo bảng `fashion_items`

Tạo bảng mới dựa trên field thực tế của `wardrobe_items`.

Pseudo SQL tham khảo, không copy y nguyên nếu schema hiện tại khác:

```sql
CREATE TABLE fashion_items (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    category_id uuid REFERENCES categories(id) ON DELETE RESTRICT,
    image_url varchar(500) NOT NULL,
    image_public_id varchar(255) NOT NULL,
    color varchar(50),
    color_hex varchar(7),
    color_hue double precision,
    color_saturation double precision,
    color_lightness double precision,
    style varchar(100),
    material varchar(100),
    pattern varchar(100),
    fit varchar(50),
    seasonality varchar(100),
    description text,
    embedding vector,
    processing_retry_count int NOT NULL DEFAULT 0,
    processing_version int NOT NULL DEFAULT 0,
    processing_started_at timestamptz,
    last_processing_attempt_at timestamptz,
    processing_error_reason text,
    review_reason varchar(100),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
```

Agent phải dựa vào kiểu dữ liệu thật trong schema hiện tại.

## Phase 2: Backfill `fashion_items` từ `wardrobe_items`

Pseudo SQL tham khảo:

```sql
INSERT INTO fashion_items (
    id,
    category_id,
    image_url,
    image_public_id,
    color,
    color_hex,
    color_hue,
    color_saturation,
    color_lightness,
    style,
    material,
    pattern,
    fit,
    seasonality,
    description,
    embedding,
    processing_retry_count,
    processing_version,
    processing_started_at,
    last_processing_attempt_at,
    processing_error_reason,
    review_reason,
    created_at,
    updated_at
)
SELECT
    id,
    category_id,
    image_url,
    image_public_id,
    color,
    color_hex,
    color_hue,
    color_saturation,
    color_lightness,
    style,
    material,
    pattern,
    fit,
    seasonality,
    description,
    embedding,
    processing_retry_count,
    processing_version,
    processing_started_at,
    last_processing_attempt_at,
    processing_error_reason,
    review_reason,
    created_at,
    updated_at
FROM wardrobe_items;
```

## Phase 3: Thêm `fashion_item_id` vào `wardrobe_items`

Pseudo SQL tham khảo:

```sql
ALTER TABLE wardrobe_items ADD COLUMN fashion_item_id uuid;

UPDATE wardrobe_items
SET fashion_item_id = id;

ALTER TABLE wardrobe_items
ALTER COLUMN fashion_item_id SET NOT NULL;

ALTER TABLE wardrobe_items
ADD CONSTRAINT wardrobe_items_fashion_item_id_fkey
FOREIGN KEY (fashion_item_id) REFERENCES fashion_items(id);
```

Có thể thêm unique index nếu business rule là một `fashion_item` chỉ nằm trong một wardrobe item:

```sql
CREATE UNIQUE INDEX wardrobe_items_fashion_item_id_key
ON wardrobe_items(fashion_item_id);
```

Agent cần xác nhận rule này trước khi thêm unique.

## Phase 4: Đổi `price` thành `purchase_price`

Nếu giữ giá cá nhân của user:

```sql
ALTER TABLE wardrobe_items RENAME COLUMN price TO purchase_price;
```

Nếu không cần giá trong MVP:

```text
Báo cáo lại trước khi drop.
```

Không được tự ý drop `price`.

## Phase 5: Chuyển `outfit_items` sang `fashion_items`

Hiện tại `outfit_items.item_id` đang trỏ `wardrobe_items.id`.

Vì `fashion_items.id` được backfill bằng old `wardrobe_items.id`, có thể đổi FK mà không đổi dữ liệu.

Pseudo migration:

```sql
ALTER TABLE outfit_items DROP CONSTRAINT outfit_items_item_id_fkey;

ALTER TABLE outfit_items RENAME COLUMN item_id TO fashion_item_id;

ALTER TABLE outfit_items
ADD CONSTRAINT outfit_items_fashion_item_id_fkey
FOREIGN KEY (fashion_item_id) REFERENCES fashion_items(id);
```

Primary key hiện tại `(outfit_id, item_id)` cần đổi theo tên mới:

```text
PRIMARY KEY (outfit_id, fashion_item_id)
```

Agent cần kiểm tra tên constraint thật trong schema hiện tại trước khi viết migration.

## Phase 6: Drop metadata khỏi `wardrobe_items` sau khi code đã chuyển

Chỉ thực hiện sau khi:

```text
- repositories đã đọc metadata từ fashion_items
- DTO/API đã join fashion_items
- search/RAG đã chuyển sang fashion_items
- outfit usecase đã dùng fashion_item_id
- tests đã pass
```

Các cột có thể drop khỏi `wardrobe_items`:

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
```

Không drop ngay trong cùng bước nếu chưa cập nhật code.

## Phase 7: Tạo bảng `digital_samples`

Pseudo schema tham khảo:

```sql
CREATE TABLE digital_samples (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    brand_id uuid NOT NULL REFERENCES brands(id),
    fashion_item_id uuid NOT NULL REFERENCES fashion_items(id),
    name varchar(255) NOT NULL,
    description text,
    target_price numeric(12,2),
    status varchar(50) NOT NULL DEFAULT 'DRAFT',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
```

Có thể thêm unique:

```sql
CREATE UNIQUE INDEX digital_samples_fashion_item_id_key
ON digital_samples(fashion_item_id);
```

Agent cần xác nhận:

```text
mỗi digital sample có đúng một fashion item hay không
```

MVP mặc định là có.

## Phase 8: Tạo bảng `digital_sample_responses`

Pseudo schema tham khảo:

```sql
CREATE TABLE digital_sample_responses (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    digital_sample_id uuid NOT NULL REFERENCES digital_samples(id) ON DELETE CASCADE,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    outfit_id uuid REFERENCES outfits(id) ON DELETE SET NULL,
    vote_type varchar(50),
    rating int,
    feedback_text text,
    created_at timestamptz NOT NULL DEFAULT now()
);
```

Agent có thể đề xuất constraint cho `rating`, nhưng không over-engineer.

## Phase 9: Thêm `outfit_source` vào `outfits`

Pseudo SQL:

```sql
ALTER TABLE outfits
ADD COLUMN outfit_source varchar(50) NOT NULL DEFAULT 'USER_CREATED';
```

Giá trị tham khảo:

```text
USER_CREATED
AI_RECOMMENDATION
DIGITAL_SAMPLE_LAB
```

Agent cần mapping với enum hiện tại nếu backend đang dùng int constants.

---

# Search, RAG, Embedding

Do metadata chuyển từ `wardrobe_items` sang `fashion_items`, agent phải cập nhật:

```text
- lexical search index
- HNSW vector index
- hybrid search query
- Elasticsearch sync worker
- AI outfit recommendation context builder
- AI chat context builder
```

## Lexical search

Hiện tại lexical search có thể đang index các field như:

```text
color
style
material
pattern
fit
seasonality
description
```

Sau migration, các field này nằm ở:

```text
fashion_items
```

Vì vậy query search phải join:

```text
wardrobe_items -> fashion_items
```

hoặc Elasticsearch document phải denormalize metadata từ `fashion_items`.

## Vector search

Nếu `embedding` là item-level embedding, chuyển index HNSW sang:

```text
fashion_items.embedding
```

Nếu hiện có index kiểu:

```text
wardrobe_items.embedding
```

thì cần tạo index mới tương ứng:

```text
fashion_items.embedding
```

Agent phải kiểm tra đúng operator class hiện tại trước khi viết migration.

---

# AI Styling flow sau migration

## B2C outfit recommendation

```text
User request outfit recommendation
→ styling gọi wardrobe contract để lấy wardrobe_items của user
→ wardrobe trả về item ownership + fashion_items metadata
→ styling build context
→ AI tạo outfit
→ wardrobe lưu outfits
→ wardrobe lưu outfit_items bằng fashion_item_id
```

## Digital Sample Lab outfit generation

```text
Brand tạo digital sample
→ tạo fashion_items
→ tạo digital_samples

User chọn digital sample để phối
→ styling lấy digital_sample + fashion_item metadata
→ styling lấy wardrobe_items + fashion_item metadata của user
→ AI tạo outfit
→ lưu outfits với outfit_source = DIGITAL_SAMPLE_LAB
→ lưu outfit_items bằng fashion_item_id
→ user vote/feedback
→ lưu digital_sample_responses
```

Quan trọng:

```text
Digital sample không nằm trong wardrobe_items.
Digital sample không tính vào wardrobe capacity.
Digital sample có thể nằm trong outfit_items thông qua fashion_item_id.
```

---

# Brand insight sau migration

Brand insight query từ:

```text
digital_samples
→ fashion_items
→ outfit_items
→ outfits
→ digital_sample_responses
```

Brand có thể xem:

```text
sample xuất hiện trong bao nhiêu outfit
sample được phối với category/style/material nào
sample được LIKE / WOULD_BUY bao nhiêu
feedback sau khi phối thử
outfit_source DIGITAL_SAMPLE_LAB tạo bao nhiêu engagement
```

Không cần:

```text
sample_outfit_trials
sample_trial_items
```

---

# API / DTO impact

Agent phải rà lại API hiện tại.

Các API wardrobe hiện tại có thể vẫn trả response giống cũ, nhưng backend sẽ build response bằng join:

```text
wardrobe_items + fashion_items
```

Ví dụ response cũ có:

```text
color
style
material
image_url
```

Sau migration, các field này không còn trực tiếp trong `wardrobe_items`, nhưng API vẫn có thể trả như cũ bằng DTO mapper.

Không để frontend bị ảnh hưởng nếu chưa cần.

---

# Những thứ không làm trong MVP

Cam kết không over-engineering:

```text
Không tạo garment_specs
Không tạo module garment riêng
Không tạo module samplelab riêng
Không tạo digital_sample_variants
Không tạo sample_outfit_trials
Không tạo sample_trial_items
Không dùng item_source_type trong outfit_items
Không dùng DIGITAL_SAMPLE_VARIANT
Không biến digital sample thành wardrobe_item
Không biến Closy thành marketplace/resale platform
```

---

# Bảng cần thêm / sửa / bỏ

## Thêm mới

```text
fashion_items
digital_samples
digital_sample_responses
```

## Sửa

```text
wardrobe_items
- thêm fashion_item_id
- đổi price thành purchase_price nếu còn dùng
- chuyển metadata sang fashion_items sau khi code sẵn sàng

outfit_items
- đổi item_id thành fashion_item_id
- đổi FK từ wardrobe_items sang fashion_items
- giữ position_x, position_y, scale, layer_order

outfits
- thêm outfit_source nếu duyệt
```

## Drop/archive theo mô hình cũ

Các bảng community/resale không còn là core:

```text
posts
post_media
comments
likes
post_items
transfer_requests
post_score_snapshots
```

Chỉ drop/archive sau khi được duyệt.

---

# Checklist cho agent khi sửa report

Agent cần cập nhật report theo checklist sau:

```text
- Nêu rõ schema hiện tại là source of truth.
- Nêu rõ field list trong doc chỉ là tham khảo.
- Chốt hướng fashion_items làm bảng item lõi.
- Chốt wardrobe_items là ownership wrapper của user.
- Chốt digital_samples là sample wrapper của brand.
- Chốt outfit_items trỏ fashion_item_id.
- Không dùng garment_specs.
- Không dùng sample_outfit_trials.
- Không dùng sample_trial_items.
- Không dùng digital_sample_variants.
- Không dùng item_source_type trong outfit_items.
- Có mapping field từ wardrobe_items hiện tại sang fashion_items/wardrobe_items mới.
- Có migration plan từng phase.
- Có tác động tới search/RAG/embedding.
- Có tác động tới API/DTO.
- Có tác động tới AI styling.
- Có tác động tới brand insight.
- Chưa code, chưa migration.
```

---

# Prompt gửi agent

Copy phần dưới đây để yêu cầu agent sửa report.

```text
Hãy cập nhật lại report rebuild theo hướng migration mới trong tài liệu này.

Quan trọng:
- Schema hiện tại trong repo/database dump là nguồn gốc.
- Field list trong tài liệu này chỉ là tham khảo để định hướng, không được copy cứng nếu khác schema hiện tại.
- Trước khi code, phải cập nhật report phân tích để mình duyệt.

Quyết định chốt:
- Không dùng garment_specs.
- Không tạo module garment riêng.
- Không tạo module samplelab riêng.
- Tạo bảng fashion_items làm item lõi dùng chung.
- wardrobe_items chỉ còn là wrapper thể hiện item thuộc tủ user.
- digital_samples là wrapper thể hiện item thử nghiệm của brand.
- Cả wardrobe_items và digital_samples đều trỏ fashion_item_id.
- outfit_items đổi từ item_id trỏ wardrobe_items sang fashion_item_id trỏ fashion_items.
- Digital sample không nằm trong wardrobe_items, không tính wardrobe capacity.
- Digital sample có thể nằm trong outfit thông qua outfit_items.fashion_item_id.
- Không dùng sample_outfit_trials.
- Không dùng sample_trial_items.
- Không dùng digital_sample_variants.
- Không dùng item_source_type trong outfit_items.
- Vote/feedback cho sample dùng bảng digital_sample_responses.
- Styling là module riêng, dùng cho cả B2C outfit recommendation và Digital Sample Lab outfit generation.

Yêu cầu report mới phải có:
- Phân tích schema hiện tại.
- Mapping field từ wardrobe_items hiện tại sang fashion_items và wardrobe_items mới.
- Schema mục tiêu.
- Migration plan từng phase.
- Tác động tới outfit_items.
- Tác động tới AI outfit recommendation.
- Tác động tới AI chat nếu có dùng item context.
- Tác động tới Digital Sample Lab.
- Tác động tới search/RAG/embedding/HNSW/lexical search.
- Tác động tới API/DTO để không làm frontend vỡ nếu chưa cần.
- Danh sách bảng thêm/sửa/drop.
- Rủi ro kỹ thuật.
- Task list chờ duyệt.

Chưa code, chưa tạo migration, chưa xoá bảng.
Chỉ cập nhật report để mình duyệt.
```
