# Closy B2B2C Rebuild Package

Tài liệu này gom lại toàn bộ định hướng quan trọng cho nhánh rebuild B2B2C của Closy, không bao gồm phần report đánh giá của agent.

Mục tiêu của file này là làm **single source of truth** để agent đọc trước khi phân tích, chỉnh báo cáo, thiết kế lại DB hoặc tiến hành refactor code.

---

# Vị trí đề xuất trong repo

Đặt file này tại:

```text
docs/project/closy_b2b2c_rebuild/rebuild_package.md
```

Cấu trúc folder đề xuất:

```text
docs/
└── project/
    └── closy_b2b2c_rebuild/
        ├── rebuild_package.md
        ├── plans/
        │   └── rebuild_plan.md
        ├── reports/
        │   └── closy_rebuild_analysis_report.md
        └── agent_prompts/
            ├── b2b2c_rebuild_review.md
            ├── module_boundary_review.md
            └── revise_rebuild_report_with_schema.md
```

Nếu muốn gọn hơn, chỉ cần giữ file package này và report hiện tại:

```text
docs/
└── project/
    └── closy_b2b2c_rebuild/
        ├── closy_b2b2c_rebuild_package.md
        └── reports/
            └── closy_rebuild_analysis_report.md
```

---

# Quyết định mô hình kinh doanh mới

Closy chuyển từ mô hình:

```text
B2C wardrobe app + AI outfit assistant + community/resale
```

sang mô hình:

```text
B2B2C fashion loyalty and co-creation platform
```

Trong mô hình mới:

- B2C vẫn là lớp người dùng: tủ đồ số, outfit, AI styling, AI chat, trải nghiệm phối đồ.
- B2B là nguồn thu chính: brand loyalty, campaign, customer service, benefit, insight và Digital Sample Lab.
- User dùng Closy để quản lý tủ đồ, phối đồ và tương tác với brand yêu thích.
- Brand dùng Closy để quản lý khách hàng thân thiết, chăm sóc sau bán hàng, chạy chiến dịch ưu đãi và test digital sample trước khi sản xuất thật.

Câu định vị:

```text
Closy is a B2B2C Fashion Loyalty & Co-creation Platform powered by digital wardrobe data and AI styling.
```

Không định vị Closy là:

```text
mini Shopee
resale platform
P2P marketplace
community social feed
Patreon for fashion
```

---

# Nguyên tắc rebuild

Đây là nhánh rebuild riêng.

Các quyết định vận hành:

- Đã backup dữ liệu local.
- Được phép thay đổi mạnh tay DB/code.
- Không cần feature flag để giữ song song mô hình cũ.
- Nếu hướng rebuild không ổn thì quay lại main.
- Agent không được code ngay khi chưa có báo cáo/plan được duyệt.
- Agent phải trả lời bằng tiếng Việt.
- Agent phải phân biệt rõ phần giữ, sửa, xoá, archive và rebuild.

---

# Kiến trúc tổng thể

Dự án là pragmatic modular monolith bằng Go/Gin.

Các nguyên tắc hiện tại cần giữ:

- Single deployable application.
- Shared infrastructure dùng chung qua DI.
- Postgres, Redis, RabbitMQ, Cloudinary, AI provider, Elasticsearch được init một lần.
- Persistent models hiện tập trung ở `internal/shared/domain/entities`.
- Cross-module communication dùng module-level `contract` package.
- Không gọi trực tiếp usecase/repository nội bộ của module khác nếu đã là cross-module dependency.
- Handler mỏng, delegate sang usecase/service.
- Worker là runtime adapter hợp lệ, được attach qua bootstrap.

---

# Cấu trúc module mục tiêu

Module mục tiêu:

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

Ý nghĩa:

| Module         | Vai trò                                                                                          |
| -------------- | ------------------------------------------------------------------------------------------------ |
| `identity`     | Auth, user, refresh token, user profile                                                          |
| `subscription` | B2C premium, quota, AI entitlement, billing phụ                                                  |
| `garment`      | Metadata quần áo dùng chung, category, garment specs                                             |
| `wardrobe`     | Tủ đồ cá nhân, wardrobe items, outfits, outfit items                                             |
| `styling`      | AI outfit recommendation, AI chat, styling engine, prompt orchestration, RAG/retrieval/reranking |
| `brand`        | Brand Portal, brand CRM, loyalty, campaign, benefit, customer service                            |
| `samplelab`    | Digital samples, sample variants/assets, sample trials, votes, feedback, insights                |

---

# Module ownership

## `garment` owns

```text
categories
garment_specs
```

`garment` trả lời câu hỏi:

```text
Đây là loại quần áo gì?
Màu gì?
Chất liệu gì?
Style gì?
Phù hợp mùa/dịp nào?
Có embedding nào dùng cho search/styling không?
```

## `wardrobe` owns

```text
wardrobe_items
outfits
outfit_items
user wardrobe lifecycle
```

`wardrobe` trả lời câu hỏi:

```text
User nào sở hữu item này?
Item này có nằm trong tủ đồ cá nhân không?
Item này đã được dùng lần cuối khi nào?
Item này có được tính vào wardrobe capacity không?
```

## `styling` owns

```text
AI outfit recommendation
AI chat
styling context builder
prompt orchestration
fashion rules / reranking
styling retrieval orchestration
conversation contexts nếu vẫn dùng cho AI chat
messages nếu vẫn dùng cho AI chat
```

`styling` trả lời câu hỏi:

```text
Dựa trên wardrobe, garment metadata và sample context, nên phối đồ như thế nào?
AI nên trả lời user ra sao?
Sample này nên phối với những item nào trong tủ user?
```

## `brand` owns

```text
brands
brand_members
brand_customers
loyalty_accounts
loyalty_point_transactions
loyalty_tiers
brand_benefits
benefit_redemptions
brand_campaigns
campaign_posts
campaign_participants
campaign_interactions
campaign_rewards
support_tickets
support_ticket_messages
return_exchange_requests
```

`brand` trả lời câu hỏi:

```text
Brand là ai?
Ai được quản trị brand?
User nào là khách hàng thân thiết?
Điểm loyalty của khách là bao nhiêu?
Campaign/benefit/support ticket thuộc brand nào?
```

## `samplelab` owns

```text
digital_samples
digital_sample_assets
digital_sample_variants
sample_outfit_trials
sample_trial_items
sample_votes
sample_feedback
sample_insights hoặc insight query layer
```

`samplelab` trả lời câu hỏi:

```text
Brand đang test mẫu digital nào?
User đã thử phối sample nào?
Sample được vote/feedback ra sao?
Insight thị hiếu thu được là gì?
```

---

# Quyết định bắt buộc về Digital Sample Lab

Digital Sample Lab có tương tác với tủ đồ người dùng, nhưng **digital sample không phải wardrobe item**.

Bắt buộc:

- Không dùng `wardrobe_items` để lưu digital sample.
- Không để digital sample tính vào wardrobe capacity.
- Không để user chỉnh sửa digital sample như item cá nhân.
- Không để brand quản lý lifecycle của `wardrobe_items`.
- Không để samplelab ghi trực tiếp vòng đời của wardrobe item.
- Samplelab chỉ tham chiếu `wardrobe_item_id` trong trial và lưu snapshot.
- Samplelab gọi styling để tạo kết quả phối thử.
- Styling lấy dữ liệu wardrobe qua `wardrobe/contract`.

Sai hướng:

```text
digital_sample = wardrobe_item
```

Đúng hướng:

```text
garment_specs
    ↑
    ├── wardrobe_items
    └── digital_samples
```

---

# Metadata hiện tại của wardrobe item

Schema hiện tại của `wardrobe_items` có các field quan trọng:

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

Cần preserve hoặc giải thích rõ nếu bỏ metadata nào.

---

# Phân loại field từ `wardrobe_items`

## Chuyển sang `garment_specs`

Các field mô tả bản chất/thông số quần áo:

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

Lý do:

- Digital sample cũng cần metadata quần áo tương tự wardrobe item.
- Styling/RAG cần dùng metadata chung cho cả item thật và sample.
- Không nên duplicate logic extract/search metadata ở hai module.

## Giữ ở `wardrobe_items`

Các field thuộc ownership/lifecycle của item user:

```text
id
user_id
garment_spec_id
image_url
image_public_id
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

Lý do:

- Đây là item user thật sự sở hữu.
- Các field này không áp dụng cho digital sample.
- Digital sample có lifecycle riêng của brand.

## Field cần quyết định rõ

```text
price
```

Cách xử lý đề xuất:

- Nếu `price` từng phục vụ resale/community marketplace, bỏ khỏi core wardrobe hoặc archive.
- Nếu muốn giữ, đổi nghĩa thành `purchase_price` optional của user-owned item.
- Với digital sample, không dùng chung `price`; dùng field riêng như `target_price`, `expected_price`, `price_range`.

## Field ảnh/asset

```text
image_url
image_public_id
```

Không nên mặc định đưa vào `garment_specs`.

Cách đề xuất:

- `wardrobe_items.image_url` là ảnh item user.
- `wardrobe_items.image_public_id` là Cloudinary public ID của ảnh item user.
- `digital_sample_assets` hoặc `digital_sample_variants` quản lý ảnh sample của brand.
- `garment_specs` có thể có `representative_image_url` nếu cần, nhưng không phải nguồn asset chính.

---

# Thiết kế DB mục tiêu

## `garment_specs`

Do module `garment` sở hữu.

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

Ghi chú:

- `metadata` JSONB dùng để chứa thông tin mở rộng chưa ổn định.
- `embedding` cần đánh giá lại loại vector hiện tại.
- Nếu embedding dùng chung cho search/styling trên item và sample, nên đặt ở `garment_specs`.
- Nếu embedding phụ thuộc ảnh cụ thể của user, cần tách thêm loại embedding hoặc giữ ở item asset.
- `garment_specs` nên gần immutable.
- Nếu metadata quan trọng đã được dùng trong outfit/trial mà cần sửa, ưu tiên tạo spec mới thay vì update trực tiếp.

## `categories`

Do module `garment` sở hữu.

```text
categories
- id
- parent_id nullable
- name
- slug
- type hoặc level nếu hiện schema đang có
- created_at
- updated_at
```

Ghi chú:

- Nếu schema hiện tại có thêm field cho category, preserve hoặc giải thích rõ nếu bỏ.
- Category không nên thuộc riêng `wardrobe`, vì digital sample cũng cần category.

## `wardrobe_items`

Do module `wardrobe` sở hữu.

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

Ghi chú:

- Không lưu digital sample ở đây.
- Không giữ metadata chính nếu đã chuyển sang `garment_specs`.
- Nếu denormalize field nào vì performance/search, phải ghi rõ rule đồng bộ.

## `outfits`

Do module `wardrobe` sở hữu.

```text
outfits
- id
- user_id
- name
- occasion
- style
- seasonality
- weather_context
- ai_generated
- is_deleted
- created_at
- updated_at
```

Field cụ thể cần đối chiếu lại schema hiện tại.

## `outfit_items`

Do module `wardrobe` sở hữu.

```text
outfit_items
- id
- outfit_id
- wardrobe_item_id
- slot
- created_at
```

Ghi chú:

- `outfit_items` nên trỏ tới `wardrobe_items`.
- Không nên dùng `outfit_items` để chứa digital sample trong trial nếu trial chưa được user save thành outfit thật.
- Nếu có nhu cầu save trial thành outfit, tạo outfit thật sau khi user confirm.

## `digital_samples`

Do module `samplelab` sở hữu.

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
- price_range_min
- price_range_max
- test_start_at
- test_end_at
- created_at
- updated_at
```

Ghi chú:

- `name` và `description` ở đây là product/concept display info.
- `garment_specs.description` là mô tả metadata quần áo.
- Có thể giữ cả hai nếu phân biệt rõ.

## `digital_sample_assets`

Do module `samplelab` sở hữu.

```text
digital_sample_assets
- id
- sample_id
- asset_url
- asset_public_id
- asset_type
- sort_order
- created_at
- updated_at
```

## `digital_sample_variants`

Do module `samplelab` sở hữu.

```text
digital_sample_variants
- id
- sample_id
- garment_spec_id nullable
- variant_name
- color
- color_hex
- size_label nullable
- image_url nullable
- image_public_id nullable
- status
- created_at
- updated_at
```

Ghi chú:

- Nếu variant chỉ khác ảnh/màu nhẹ thì có thể dùng asset metadata.
- Nếu variant khác metadata thời trang đáng kể, cho variant trỏ `garment_spec_id` riêng.

## `sample_outfit_trials`

Do module `samplelab` sở hữu.

```text
sample_outfit_trials
- id
- sample_id
- sample_variant_id nullable
- user_id
- styling_result_snapshot
- sample_spec_snapshot
- saved_outfit_id nullable
- created_at
```

Ghi chú:

- Không bắt buộc có `outfit_id`.
- `saved_outfit_id` chỉ có khi user quyết định save trial thành outfit thật.
- Trial phải lưu snapshot để giữ lịch sử ngay cả khi sample/item/spec thay đổi.

## `sample_trial_items`

Do module `samplelab` sở hữu.

```text
sample_trial_items
- id
- trial_id
- wardrobe_item_id
- wardrobe_item_snapshot
- garment_spec_snapshot
- slot
- created_at
```

## `sample_votes`

Do module `samplelab` sở hữu.

```text
sample_votes
- id
- sample_id
- sample_variant_id nullable
- user_id
- vote_type
- rating nullable
- created_at
```

## `sample_feedback`

Do module `samplelab` sở hữu.

```text
sample_feedback
- id
- sample_id
- sample_variant_id nullable
- user_id
- feedback_text
- feedback_metadata
- created_at
```

## Brand core tables

```text
brands
brand_members
brand_customers
```

## Loyalty tables

```text
loyalty_accounts
loyalty_point_transactions
loyalty_tiers
brand_benefits
benefit_redemptions
```

## Campaign tables

```text
brand_campaigns
campaign_posts
campaign_participants
campaign_interactions
campaign_rewards
```

## Customer service tables

```text
support_tickets
support_ticket_messages
return_exchange_requests
```

---

# Contract giữa các module

## `garment/contract`

```text
CreateGarmentSpec(input)
GetGarmentSpec(id)
GetGarmentSpecs(ids)
CloneGarmentSpec(id, overrides)
LockGarmentSpec(id)
ResolveCategory(id)
SearchGarmentSpecs(query)
```

## `wardrobe/contract`

```text
ListUserWardrobeItemsForStyling(userID, filter)
GetWardrobeItemForStyling(userID, itemID)
GetUserOutfitHistory(userID, filter)
GetUserStyleProfile(userID)
```

Return DTO không expose entity nội bộ tùy tiện. DTO nên chứa:

```text
wardrobe_item_id
garment_spec_id
image_url
last_used_at
slot/category info cần cho styling
```

## `styling/contract`

```text
RecommendOutfit(input)
ChatWithStylingAssistant(input)
GenerateSampleTrialStyling(input)
BuildStylingContext(input)
```

`styling` được gọi bởi:

- User-facing AI outfit recommendation API.
- User-facing AI chat API.
- `samplelab` khi cần thử phối digital sample với tủ đồ user.

## `samplelab/contract`

```text
GetDigitalSampleForStyling(sampleID)
RecordSampleTrial(input)
RecordSampleVote(input)
RecordSampleFeedback(input)
GetSampleInsight(sampleID)
```

## `subscription/contract`

```text
CanUseAI(userID, operation)
ReserveAIUsage(userID, operation)
FinalizeAIUsage(reservationID)
RefundAIUsage(reservationID)
ConsumeAIQuota(userID, operation)
```

Quy tắc:

- `styling` không sở hữu billing/quota.
- `styling` gọi `subscription/contract` để check/reserve/finalize/refund quota.
- Shared AI provider vẫn ở `internal/shared/infrastructure/ai`.

---

# Luồng chính

## User tạo wardrobe item

```text
User upload image
→ wardrobe receives upload
→ shared media stores image
→ AI extraction pipeline extracts metadata
→ garment.CreateGarmentSpec(metadata)
→ wardrobe.CreateWardrobeItem(user_id, garment_spec_id, image_url, image_public_id)
```

## User nhận outfit recommendation

```text
User requests outfit recommendation
→ styling checks quota via subscription contract
→ styling gets wardrobe context via wardrobe contract
→ styling gets garment specs via garment contract
→ styling builds prompt/retrieval/reranking context
→ shared AI provider returns recommendation
→ styling finalizes quota
→ response returned to user
```

## User chat với AI stylist

```text
User sends chat message
→ styling checks quota via subscription contract
→ styling loads conversation context
→ styling loads wardrobe/garment context if needed
→ shared AI provider generates response
→ styling stores message/context
→ styling finalizes quota
```

## Brand tạo digital sample

```text
Brand staff creates sample
→ brand permission checked via brand_members
→ samplelab receives sample metadata and assets
→ garment.CreateGarmentSpec(metadata)
→ samplelab.CreateDigitalSample(brand_id, garment_spec_id)
→ samplelab stores digital_sample_assets / variants if any
```

## User thử digital sample với tủ đồ

```text
User opens digital sample
→ samplelab loads digital sample
→ samplelab calls styling.GenerateSampleTrialStyling
→ styling loads user wardrobe via wardrobe contract
→ styling loads sample garment spec via garment contract
→ styling generates trial result
→ samplelab stores sample_outfit_trials
→ samplelab stores sample_trial_items snapshots
→ user can vote/feedback
```

---

# Search, RAG và embedding

Agent phải đánh giá lại HNSW/vector index hiện tại.

Câu hỏi cần trả lời:

- `embedding` hiện đại diện cho image embedding, text metadata embedding hay combined embedding?
- Nếu embedding dùng cho semantic retrieval của clothing metadata, nên chuyển sang `garment_specs.embedding`.
- Nếu embedding phụ thuộc ảnh cụ thể của user, có thể cần giữ embedding ở `wardrobe_items` hoặc tạo bảng asset embedding riêng.
- Digital samples cũng cần được search/rank trong sample trial/styling.
- Cần đảm bảo RAG có thể lấy cả wardrobe item và digital sample context mà không làm prompt phình quá lớn.

Đề xuất mặc định:

```text
garment_specs.embedding = embedding metadata/style dùng chung
wardrobe_items.image_url/image_public_id = asset user item
digital_sample_assets = asset brand sample
```

Nếu cần nhiều vector:

```text
garment_specs.text_embedding
garment_specs.image_embedding nullable
digital_sample_assets.image_embedding nullable
```

Nhưng không over-engineering nếu MVP chưa cần.

---

# AI cost/quota boundary

Quyết định:

```text
subscription owns quota/entitlement
styling consumes quota through subscription contract
shared infrastructure owns AI provider client
styling owns prompt orchestration and AI usecase
```

Không chuyển toàn bộ AI cost control sang `styling` nếu nó là entitlement/quota của user.

`styling` chỉ nên gọi:

```text
CanUseAI
ReserveAIUsage
FinalizeAIUsage
RefundAIUsage
```

hoặc cơ chế tương đương.

---

# Community/resale cần archive hoặc drop

Các phần sau không còn là core:

```text
community module
posts như social feed tự do
comments/likes nếu chỉ phục vụ social feed
post_score_snapshots
post_items
transfer_requests
buyer/seller flow
P2P resale
hot feed/ranking bài post
```

Có thể rebuild lại một phần dưới module `brand` nếu phục vụ campaign:

```text
posts → campaign_posts
likes/comments → campaign_interactions
```

Nhưng không giữ nghĩa cũ là feed cộng đồng tự do.

---

# API định hướng mới

## User-facing B2C

```text
GET /api/v1/brands
POST /api/v1/brands/:brandId/join-loyalty
GET /api/v1/me/brand-loyalties

GET /api/v1/digital-samples
GET /api/v1/digital-samples/:sampleId
POST /api/v1/digital-samples/:sampleId/votes
POST /api/v1/digital-samples/:sampleId/feedback
POST /api/v1/digital-samples/:sampleId/outfit-trials

POST /api/v1/ai/outfit-recommendations
POST /api/v1/ai/chat
```

## Brand Portal

```text
GET /api/v1/brand-portal/brands/:brandId/customers
POST /api/v1/brand-portal/brands/:brandId/loyalty/accounts/:accountId/adjust-points

POST /api/v1/brand-portal/brands/:brandId/campaigns
GET /api/v1/brand-portal/brands/:brandId/campaigns
POST /api/v1/brand-portal/brands/:brandId/benefits
GET /api/v1/brand-portal/brands/:brandId/support-tickets
POST /api/v1/brand-portal/brands/:brandId/support-tickets

POST /api/v1/brand-portal/brands/:brandId/digital-samples
GET /api/v1/brand-portal/digital-samples/:sampleId/insights
```

---

# Phase triển khai đề xuất

## Phase chuẩn bị

- Đọc file package này.
- Đọc schema hiện tại.
- Đọc report hiện tại nếu có.
- Không code ngay.
- Cập nhật report thiết kế để phản ánh đúng metadata và module boundaries.

## Phase làm sạch mô hình cũ

- Archive/xoá module community khỏi MVP mới.
- Drop/archive bảng resale/community cũ sau khi được duyệt.
- Gỡ route posts/transfers khỏi API mới.

## Phase tách `garment`

- Chuyển `categories` sang garment ownership.
- Tạo `garment_specs`.
- Mapping metadata từ `wardrobe_items` hiện tại sang `garment_specs`.
- Đánh giá lại embedding/HNSW index.

## Phase tách `wardrobe`

- `wardrobe_items` chỉ giữ ownership/lifecycle.
- Thêm `garment_spec_id`.
- Giữ outfit/outfit_items cho user-owned outfits.
- Expose `wardrobe/contract` cho styling/samplelab.

## Phase tách `styling`

- Di chuyển AI outfit recommendation khỏi wardrobe.
- Di chuyển AI chat khỏi wardrobe nếu hiện đang nằm trong wardrobe.
- `styling` đọc wardrobe/garment qua contract.
- `styling` gọi subscription contract cho quota.
- Shared AI provider vẫn ở shared infrastructure.

## Phase xây `brand`

- Brand profile.
- Brand members/roles.
- Brand customers.
- Loyalty account/transaction/tier.
- Campaign/benefit/support MVP.

## Phase xây `samplelab`

- Digital samples.
- Sample assets/variants.
- Trial/vote/feedback.
- Insight query cơ bản.
- Gọi styling để tạo sample trial.

## Phase seed/demo/test

- Seed 1 brand demo.
- Seed brand members.
- Seed loyalty tiers/benefits.
- Seed digital samples.
- Demo user thử sample với wardrobe.
- Demo vote/feedback và brand xem insight.

---

# Câu hỏi cần xác nhận

Agent phải hỏi lại nếu chưa rõ:

- MVP Digital Sample Lab chỉ dùng ảnh 2D hay cần 3D/avatar viewer?
- Loyalty offline purchase do staff cộng tay hay tích hợp POS/hóa đơn?
- `price` trong wardrobe item nên bỏ, đổi thành `purchase_price`, hay giữ để hiển thị cá nhân?
- `embedding` hiện là loại embedding nào?
- Có cần giữ conversation context/message trong `styling` hay chuyển một phần sang shared AI chat context?
- Có cần migrate data cũ hay tạo baseline schema mới trên nhánh rebuild?

---

# Prompt dùng để gửi agent

Copy phần dưới đây để gửi agent.

```text
Bạn đang làm việc trên nhánh rebuild B2B2C của dự án Closy.

Hãy trả lời hoàn toàn bằng tiếng Việt.

Trước khi code, hãy đọc kỹ file sau:

docs/project/closy_b2b2c_rebuild/closy_b2b2c_rebuild_package.md

Nếu có report hiện tại, hãy đọc thêm:

docs/project/closy_b2b2c_rebuild/reports/closy_rebuild_analysis_report.md

Mục tiêu hiện tại: cập nhật lại báo cáo thiết kế/rebuild theo đúng package này, chưa code, chưa tạo migration, chưa xoá module.

Yêu cầu:

- Không dùng wardrobe_items để lưu digital sample.
- Không để digital sample tính vào wardrobe capacity.
- Không để samplelab quản lý lifecycle của wardrobe_items.
- Không để styling nằm sâu trong wardrobe nữa.
- Không để styling gọi trực tiếp repository của wardrobe.
- Styling phải giao tiếp qua contract của wardrobe, garment, samplelab nếu cần.
- Styling phải check/consume quota qua subscription contract.
- Shared AI provider vẫn nằm ở shared infrastructure.
- garment module quản lý garment_specs và categories.
- wardrobe_items và digital_samples đều tham chiếu garment_spec_id.
- Các metadata hiện có trong wardrobe_items phải được preserve hoặc giải thích rõ nếu bỏ.
- Phải đánh giá tác động tới embedding/HNSW index/search/RAG.
- Phải giải thích cách xử lý price.
- Không giữ community/resale/social feed làm core product.
- Không biến Closy thành mini Shopee, marketplace hoặc P2P resale platform.

Hãy trả về báo cáo tiếng Việt gồm:

- Tóm tắt quyết định kiến trúc mới
- Những điểm report cũ còn đúng
- Những điểm report cũ cần sửa
- Phân loại field hiện tại của wardrobe_items
- Thiết kế module cuối cùng
- Thiết kế database cuối cùng
- Thiết kế contract giữa các module
- Luồng xử lý chính sau khi tách module
- Ảnh hưởng tới AI outfit recommendation
- Ảnh hưởng tới AI chat
- Ảnh hưởng tới Digital Sample Lab
- Ảnh hưởng tới search/RAG/embedding
- Ảnh hưởng tới subscription/quota/cost control
- Danh sách bảng cần giữ
- Danh sách bảng cần drop/archive
- Danh sách bảng cần thêm mới
- Danh sách API cần giữ
- Danh sách API cần xoá/archive
- Danh sách API cần thêm mới
- Rủi ro kỹ thuật
- Kế hoạch phase triển khai
- Danh sách task cụ thể để chờ mình duyệt

Không code ngay. Chỉ cập nhật báo cáo và chờ duyệt.
```
