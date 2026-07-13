# Prompt cho agent cập nhật báo cáo rebuild theo schema hiện tại

Bạn đang làm việc trên nhánh rebuild B2B2C của dự án Closy.

Hãy trả lời hoàn toàn bằng tiếng Việt.

## Bối cảnh

Dự án Closy đang chuyển từ mô hình B2C wardrobe app + community/resale sang mô hình B2B2C:

- B2C là ứng dụng tủ đồ số và AI outfit assistant cho người dùng.
- B2B là nguồn thu chính thông qua Brand Loyalty, Campaign, Customer Service, Benefit, Insight và Digital Sample Lab.
- Community feed, resale, transfer item, mini marketplace và P2P transaction không còn là core product.
- Dự án là pragmatic modular monolith bằng Go/Gin.
- Cross-module communication dùng module-level `contract` package, ví dụ `internal/modules/wardrobe/contract`, không gọi trực tiếp usecase/repository nội bộ của module khác.
- Persistent models hiện nằm tập trung trong `internal/shared/domain/entities`.
- Shared infrastructure như DB, Redis, RabbitMQ, Cloudinary, AI provider và Elasticsearch vẫn được dùng chung qua DI.

## Tài liệu cần đọc trước

Trước khi trả lời, hãy đọc lại các tài liệu hiện có trong repo:

```text
docs/project/b2b2c-rebuild-plan.md
docs/development/agent-prompts/b2b2c-rebuild-review.md
docs/system-design/module-boundaries-garment-wardrobe-samplelab.md
docs/system-design/architecture.md
```

Nếu file architecture rule đang nằm tên khác, hãy tìm tài liệu chứa nội dung về pragmatic modular monolith, shared entities, module contract, shared infrastructure và routing.

## Mục tiêu của task này

Không code ngay.

Hãy cập nhật lại báo cáo phân tích/rebuild trước đó, vì thiết kế hiện tại đang quên một số metadata quan trọng của `wardrobe_items` trong schema hiện tại.

Báo cáo mới phải điều chỉnh lại thiết kế các module:

```text
internal/modules/
├── identity/
├── subscription/
├── garment/
├── wardrobe/
├── styling/
├── brand/
└── samplelab/
```

Trong đó:

- `garment` quản lý metadata quần áo dùng chung, đặc biệt là `garment_specs` và `categories`.
- `wardrobe` quản lý item người dùng sở hữu, outfit cá nhân và trạng thái sử dụng tủ đồ.
- `styling` quản lý AI outfit recommendation, AI chat, styling engine, prompt orchestration, RAG/retrieval/reranking nếu có.
- `brand` quản lý Brand Portal, CRM, loyalty, campaign, benefit và customer service.
- `samplelab` quản lý digital sample, sample assets, sample variants, outfit trials, votes, feedback và insight.
- `samplelab` gọi `styling` qua contract khi cần tạo kết quả phối thử.
- `styling` đọc dữ liệu wardrobe/garment/sample thông qua contracts, không gọi trực tiếp repository nội bộ của module khác.

## Schema hiện tại cần dùng làm nguồn tham chiếu

Trong schema hiện tại, `wardrobe_items` đang có các field quan trọng sau:

```text
wardrobe_items
- id
- user_id
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
- price
- status
- item_type
- embedding
- last_used_at
- processing_retry_count
- processing_version
- processing_started_at
- last_processing_attempt_at
- processing_error_reason
- review_reason
- is_deleted
- created_at
- updated_at
```

Trong đó cần phân loại lại cẩn thận:

### Metadata quần áo nên đưa sang `garment_specs`

Các field này thể hiện bản chất/thông số thời trang của một item và có thể dùng chung cho cả `wardrobe_items` lẫn `digital_samples`:

```text
category_id
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
```

Lưu ý:

- `embedding` hiện nằm trong `wardrobe_items` và có HNSW index. Khi tách `garment_specs`, cần đánh giá có nên chuyển embedding sang `garment_specs` để dùng chung cho wardrobe item và digital sample hay không.
- Nếu có nhiều loại embedding, hãy đề xuất rõ: text/style embedding, image embedding, taste embedding.
- Nếu giữ embedding ở `wardrobe_items`, hãy giải thích vì sao. Nếu chuyển sang `garment_specs`, hãy nêu tác động tới search/RAG/styling.

### Field ảnh/asset cần tách khỏi metadata lõi

Các field này là asset của item hiện tại, nhưng cần thiết kế lại để dùng được cho cả wardrobe và sample:

```text
image_url
image_public_id
```

Yêu cầu đánh giá:

- Với `wardrobe_items`, ảnh là ảnh item của user.
- Với `digital_samples`, ảnh là asset hoặc concept image của brand.
- Không bắt buộc đưa ảnh vào `garment_specs`, nhưng có thể có `representative_image_url` nếu cần.
- Nếu digital sample có nhiều ảnh/màu/variant, nên thiết kế bảng `digital_sample_assets` hoặc `digital_sample_variants`.

### Field thuộc về lifecycle của wardrobe item

Các field này không thuộc `garment_specs`, chỉ thuộc `wardrobe_items`:

```text
user_id
status
item_type
last_used_at
processing_retry_count
processing_version
processing_started_at
last_processing_attempt_at
processing_error_reason
review_reason
is_deleted
created_at
updated_at
```

Yêu cầu:

- Không đưa các field này sang `digital_samples`.
- Không để digital sample tính vào wardrobe capacity.
- Không để brand quản lý lifecycle của `wardrobe_items`.
- Không để user sửa digital sample như item trong tủ đồ cá nhân.

### Field cần đánh giá lại

```text
price
```

Yêu cầu:

- Phân tích `price` hiện tại có còn cần cho wardrobe item không.
- Nếu `price` từng phục vụ resale/community marketplace thì nên bỏ khỏi `wardrobe_items` core hoặc chuyển nghĩa thành optional purchase price của user-owned item.
- Với `digital_samples`, nếu brand cần giá dự kiến thì dùng field riêng như `target_price`, `expected_price` hoặc `price_range`, không dùng chung logic resale.

## Thiết kế DB mong muốn cần đánh giá và hoàn thiện

Hãy đề xuất lại schema theo hướng sau, nhưng được phép chỉnh nếu có lý do tốt.

### `garment_specs`

`garment_specs` là bảng metadata quần áo trung lập do module `garment` sở hữu.

Gợi ý field:

```text
garment_specs
- id
- category_id
- name
- description
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
- metadata
- embedding
- version
- is_locked
- created_at
- updated_at
```

Yêu cầu:

- Phải phản ánh lại đầy đủ các metadata hiện có của `wardrobe_items`.
- Có thể đề xuất thêm field như `style_tags`, `season_tags`, `occasion_tags`, `color_palette`, nhưng phải phân biệt field mới với field hiện có.
- `garment_specs` nên gần immutable. Nếu metadata quan trọng đã được dùng trong outfit/trial mà cần sửa, ưu tiên tạo spec mới thay vì update trực tiếp.
- Nếu cần versioning, đề xuất rõ cách dùng `version`, `is_locked` hoặc `source_type`.

### `wardrobe_items`

`wardrobe_items` chỉ đại diện cho item người dùng thật sự sở hữu hoặc đã xác nhận trong tủ đồ cá nhân.

Gợi ý field mới:

```text
wardrobe_items
- id
- user_id
- garment_spec_id
- image_url
- image_public_id
- source_type
- status
- item_type
- last_used_at
- processing_retry_count
- processing_version
- processing_started_at
- last_processing_attempt_at
- processing_error_reason
- review_reason
- is_deleted
- created_at
- updated_at
```

Yêu cầu:

- Không còn giữ các metadata chính như color/material/style trực tiếp nếu đã chuyển sang `garment_specs`.
- Nếu cần giữ denormalized field để search/performance, phải ghi rõ field nào, lý do, và rule đồng bộ.
- `wardrobe_items` không được dùng cho digital sample.

### `digital_samples`

`digital_samples` chỉ đại diện cho mẫu thử số hóa của brand, không phải item của user.

Gợi ý field:

```text
digital_samples
- id
- brand_id
- garment_spec_id
- sample_code
- name
- description
- status
- testing_status
- launch_status
- target_price
- test_start_at
- test_end_at
- created_at
- updated_at
```

Yêu cầu:

- `digital_samples` phải tham chiếu `garment_spec_id`.
- Không dùng `wardrobe_items` để lưu sample.
- Nếu sample có nhiều màu/ảnh/variant, đề xuất thêm `digital_sample_variants` hoặc `digital_sample_assets`.
- Nếu `name` và `description` đã có ở `garment_specs`, hãy phân biệt rõ cái nào là product/concept display info, cái nào là garment metadata.

### `sample_outfit_trials`

Không nên bắt buộc tạo `outfit` thật trong bảng `outfits` khi user chỉ thử sample.

Gợi ý:

```text
sample_outfit_trials
- id
- sample_id
- user_id
- styling_result_snapshot
- sample_spec_snapshot
- created_at
```

Hoặc nếu cần lưu lại outfit thật sau khi user bấm save:

```text
sample_outfit_trials
- saved_outfit_id nullable
```

Yêu cầu:

- Không bắt buộc `outfit_id`.
- Trial phải lưu snapshot để không mất lịch sử nếu wardrobe item hoặc digital sample thay đổi.
- Nếu trial dùng nhiều wardrobe item của user, tạo thêm bảng:

```text
sample_trial_items
- id
- trial_id
- wardrobe_item_id
- wardrobe_item_snapshot
- slot
- created_at
```

## Thiết kế module và contract cần cập nhật

Hãy đề xuất contract cần có giữa các module.

### `garment/contract`

Gợi ý:

```text
CreateGarmentSpec
GetGarmentSpec
GetGarmentSpecs
CloneGarmentSpec
LockGarmentSpec
ResolveCategory
```

### `wardrobe/contract`

Gợi ý:

```text
ListUserWardrobeItemsForStyling
GetWardrobeItemForStyling
GetUserOutfitHistory
GetUserStyleProfile
```

### `styling/contract`

Gợi ý:

```text
RecommendOutfit
ChatWithStylingAssistant
GenerateSampleTrialStyling
BuildStylingContext
```

### `samplelab/contract`

Gợi ý:

```text
GetDigitalSampleForStyling
RecordSampleTrial
RecordSampleVote
RecordSampleFeedback
```

### `subscription/contract`

Gợi ý:

```text
CanUseAI
ConsumeAIQuota
ReserveAIUsage
FinalizeAIUsage
RefundAIUsage
```

Yêu cầu:

- `styling` không được gọi trực tiếp `wardrobe` repository.
- `styling` không sở hữu billing/quota.
- `styling` check/consume quota thông qua `subscription/contract`.
- Shared AI provider vẫn nằm ở `internal/shared/infrastructure/ai`.
- `samplelab` không quản lý `wardrobe_items`, chỉ tham chiếu và lưu snapshot trong trial.

## Cần cập nhật lại báo cáo theo các phần sau

Hãy trả về một báo cáo tiếng Việt có cấu trúc rõ ràng:

```text
Tóm tắt quyết định kiến trúc mới
Những điểm báo cáo cũ còn đúng
Những điểm báo cáo cũ cần sửa
Phân loại field hiện tại của wardrobe_items
Thiết kế module mới cuối cùng
Thiết kế database mới cuối cùng
Thiết kế contract giữa các module
Luồng xử lý chính sau khi tách module
Ảnh hưởng tới AI outfit recommendation
Ảnh hưởng tới AI chat
Ảnh hưởng tới Digital Sample Lab
Ảnh hưởng tới search/RAG/embedding
Ảnh hưởng tới subscription/quota/cost control
Danh sách bảng cần giữ
Danh sách bảng cần drop/archive
Danh sách bảng cần thêm mới
Danh sách API cần giữ
Danh sách API cần xoá/archive
Danh sách API cần thêm mới
Rủi ro kỹ thuật
Kế hoạch phase triển khai
Danh sách task cụ thể để chờ duyệt
```

## Các quyết định bắt buộc

Hãy tuân thủ các quyết định sau:

- Không dùng `wardrobe_items` để lưu digital sample.
- Không biến Closy thành marketplace, mini Shopee hoặc P2P resale platform.
- Không giữ `community` làm core module.
- Không để `styling` nằm sâu trong `wardrobe`.
- Không để `styling` gọi trực tiếp repository của `wardrobe`.
- Không để `samplelab` quản lý lifecycle của `wardrobe_items`.
- Không để `brand` quản lý `garment_specs`.
- Không để `wardrobe` quản lý `garment_specs`.
- `garment_specs` do module `garment` quản lý.
- `wardrobe_items` và `digital_samples` đều tham chiếu `garment_spec_id`.
- Các metadata hiện có của `wardrobe_items` phải được preserve hoặc giải thích rõ lý do bỏ.
- Phải chỉ rõ tác động tới HNSW vector index và lexical search hiện tại.
- Phải chỉ rõ cách xử lý `price`.

## Lưu ý về output

Không code ngay.

Không tạo migration ngay.

Không xoá module ngay.

Chỉ cập nhật báo cáo thiết kế và đưa ra plan để mình duyệt.

Nếu bạn thấy có điểm nào chưa chắc chắn, hãy nêu rõ ở phần "Câu hỏi cần xác nhận", không tự đoán.
