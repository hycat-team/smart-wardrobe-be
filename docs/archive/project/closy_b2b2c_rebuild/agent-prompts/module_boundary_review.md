# Tách bạch module Garment, Wardrobe, Brand và Digital Sample Lab

## Mục đích

Tài liệu này bổ sung cho kế hoạch rebuild Closy theo mô hình B2B2C. Mục tiêu là làm rõ ranh giới module khi hệ thống có cả:

- Tủ đồ cá nhân của người dùng.
- Mẫu thử số hoá của brand trong Digital Sample Lab.
- Metadata quần áo dùng chung như category, màu sắc, chất liệu, pattern, style tags, season tags, occasion tags.
- Cơ chế giao tiếp giữa module thông qua tầng `contract`.

Vấn đề chính cần xử lý là: **digital sample có metadata giống wardrobe item, nhưng không được xem là wardrobe item**.

Kết luận đề xuất:

> Tạo module trung lập `garment` để sở hữu `garment_specs` và metadata quần áo dùng chung.  
> `wardrobe` sở hữu item người dùng thật sự có trong tủ đồ.  
> `samplelab` sở hữu digital sample, trial, vote, feedback và snapshot thử phối.  
> Các module chỉ giao tiếp qua `contract`, không import trực tiếp use case nội bộ của nhau.

---

## Bối cảnh kiến trúc hiện tại

Backend Closy là một **pragmatic modular monolith** bằng Go, chạy trong một Gin HTTP service duy nhất.

Một số rule hiện tại cần giữ:

- Persistent GORM entities tập trung trong `internal/shared/domain/entities`.
- Shared infrastructure như DB, Redis, RabbitMQ, Cloudinary, AI provider và Elasticsearch được khởi tạo một lần và dùng qua DI.
- Cross-module communication dùng package `contract` ở cấp module, ví dụ `internal/modules/identity/contract`.
- Không áp dụng DDD thuần với aggregate root, value object, domain event bắt buộc.
- Module boundary có thật nhưng không cần mọi module phải có cấu trúc provider giống nhau tuyệt đối.
- Handler và worker là presentation adapter; business execution nằm ở use case/application layer.
- Không import trực tiếp use case implementation của module khác nếu đã có contract phù hợp.

Vì vậy, trong tài liệu này, từ “sở hữu bảng” không có nghĩa là entity Go phải nằm trong module đó. Do project dùng shared GORM entity pool, “sở hữu” được hiểu là:

- Module nào định nghĩa vòng đời nghiệp vụ chính của bảng.
- Module nào có quyền create/update/delete hoặc thay đổi trạng thái chính.
- Module nào expose contract để module khác đọc hoặc yêu cầu thao tác.
- Module nào chịu trách nhiệm migration/schema design tương ứng trong rebuild.

---

## Vấn đề cần tránh

Không thiết kế digital sample như một bản ghi trong `wardrobe_items`.

Lý do:

- `wardrobe_items` đại diện cho item người dùng thật sự sở hữu hoặc đã xác nhận trong tủ đồ cá nhân.
- `digital_samples` đại diện cho mẫu thử số hoá thuộc brand, có thể chưa sản xuất thật.
- Digital sample không được tính vào wardrobe capacity của user.
- User không được sửa hoặc xoá digital sample như item cá nhân.
- Brand không được quản lý vòng đời của `wardrobe_items`.
- AI outfit thường ngày không nên tự dùng digital sample như đồ user đang sở hữu.
- Digital sample có lifecycle khác: draft, testing, archived, production approved.
- Wardrobe item có lifecycle khác: uploaded, confirmed, active, archived, deleted.

Điểm giống nhau giữa hai loại này là **metadata quần áo**, không phải ownership hay lifecycle.

---

## Quyết định module boundary

Cấu trúc module đề xuất:

```text
internal/modules/
├── identity/
├── subscription/
├── garment/
├── wardrobe/
├── brand/
└── samplelab/
```

Nếu agent vẫn chọn tách nhỏ hơn cho loyalty/campaign/support thì có thể đề xuất trong báo cáo riêng, nhưng tài liệu này tập trung vào ranh giới giữa `garment`, `wardrobe` và `samplelab`.

---

## Trách nhiệm của từng module

| Module | Sở hữu nghiệp vụ | Không được làm |
|---|---|---|
| `garment` | Metadata quần áo dùng chung, `garment_specs`, category, clothing taxonomy nếu rebuild | Không quản lý item của user, không quản lý sample campaign, không quản lý loyalty |
| `wardrobe` | `wardrobe_items`, `outfits`, `outfit_items`, hành vi tủ đồ cá nhân, AI outfit trên item user sở hữu | Không quản lý digital sample như item cá nhân, không sửa trực tiếp sample |
| `brand` | `brands`, `brand_members`, `brand_customers`, brand portal, loyalty, campaign, benefit, support nếu gom trong brand module | Không quản lý `garment_specs` trực tiếp, không quản lý vòng đời wardrobe item |
| `samplelab` | `digital_samples`, sample assets, outfit trials, trial items, votes, feedback, insight snapshots | Không sở hữu `wardrobe_items`, không tạo item trong tủ đồ trừ khi có flow mua/xác nhận riêng ở tương lai |

---

## Module Garment

### Vai trò

`garment` là module trung lập quản lý mô tả quần áo dùng chung cho nhiều nghiệp vụ.

Nó trả lời câu hỏi:

> Món đồ hoặc mẫu thiết kế này là loại quần áo gì, màu gì, chất liệu gì, phong cách gì, phù hợp mùa/dịp nào?

Nó không trả lời câu hỏi:

> Ai sở hữu món này?  
> User đã mặc món này chưa?  
> Brand đang test sample này ở campaign nào?  
> Mẫu này có được production approved chưa?

### Bảng đề xuất

```text
garment_specs
- id
- category_id
- name
- description
- dominant_color
- color_palette
- material
- pattern
- season_tags
- style_tags
- occasion_tags
- formality
- fit
- metadata
- version
- is_locked
- source_type
- created_by_user_id
- created_by_brand_id
- created_at
- updated_at
```

`source_type` có thể dùng các giá trị như:

```text
USER_UPLOAD
BRAND_SAMPLE
AI_GENERATED
MANUAL
```

`created_by_user_id` và `created_by_brand_id` có thể nullable. Không dùng các field này để xác định quyền sở hữu item thật; chúng chỉ mô tả nguồn tạo metadata.

### Category

Nếu rebuild mạnh tay, nên chuyển `categories` sang phạm vi `garment` thay vì để riêng trong `wardrobe`.

Lý do:

- Wardrobe item cần category.
- Digital sample cũng cần category.
- AI styling và outfit recommendation cũng dùng category.
- Category là taxonomy quần áo, không phải nghiệp vụ riêng của tủ đồ.

---

## Module Wardrobe

### Vai trò

`wardrobe` sở hữu tủ đồ cá nhân của user.

Nó trả lời câu hỏi:

> User nào sở hữu item này?  
> Item này có nằm trong tủ đồ không?  
> Item này có tính vào capacity không?  
> Item này đã mặc bao nhiêu lần?  
> Item này có được dùng trong outfit cá nhân không?

### Bảng đề xuất

```text
wardrobe_items
- id
- user_id
- garment_spec_id
- image_url
- source_type
- status
- wear_count
- last_worn_at
- is_favorite
- confirmed_at
- created_at
- updated_at
```

```text
outfits
- id
- user_id
- name
- occasion
- style
- status
- created_at
- updated_at
```

```text
outfit_items
- id
- outfit_id
- wardrobe_item_id
- slot
- created_at
```

### Rule quan trọng

`wardrobe_items` chỉ chứa item user thật sự sở hữu hoặc đã xác nhận trong tủ đồ.

Không lưu digital sample vào `wardrobe_items`.

Không để brand update trực tiếp `wardrobe_items`.

Không tính digital sample vào wardrobe capacity.

---

## Module Sample Lab

### Vai trò

`samplelab` sở hữu Digital Sample Lab.

Nó trả lời câu hỏi:

> Brand đang test mẫu thiết kế nào?  
> User nào đã thử phối sample này?  
> Sample này được vote như thế nào?  
> Sample này thường được phối với item gì trong tủ đồ user?  
> Brand nhận được insight gì từ các trial?

### Bảng đề xuất

```text
digital_samples
- id
- brand_id
- garment_spec_id
- collection_id
- sample_code
- sample_image_url
- testing_status
- launch_status
- target_price
- test_start_at
- test_end_at
- created_at
- updated_at
```

```text
sample_assets
- id
- sample_id
- asset_url
- asset_type
- sort_order
- created_at
```

```text
sample_outfit_trials
- id
- sample_id
- brand_id
- user_id
- sample_spec_snapshot
- trial_result_snapshot
- created_at
```

```text
sample_trial_items
- id
- trial_id
- wardrobe_item_id
- wardrobe_item_spec_snapshot
- slot
- created_at
```

```text
sample_votes
- id
- sample_id
- user_id
- vote_type
- rating
- created_at
```

```text
sample_feedback
- id
- sample_id
- user_id
- content
- sentiment
- created_at
```

### Rule quan trọng

`samplelab` được phép tham chiếu `wardrobe_item_id` để biết item nào đã được dùng trong trial.

`samplelab` không được sở hữu hoặc thay đổi vòng đời `wardrobe_items`.

`samplelab` phải lưu snapshot metadata tại thời điểm trial để bảo toàn lịch sử.

---

## Vì sao cần snapshot trong Sample Lab

Ngay cả khi `wardrobe_items` và `digital_samples` cùng tham chiếu `garment_specs`, trial vẫn phải lưu snapshot.

Lý do:

- User có thể xoá wardrobe item sau khi trial.
- Brand có thể archive hoặc sửa sample bằng spec mới.
- Garment metadata có thể được chuẩn hoá lại trong tương lai.
- Brand insight cần phản ánh hành vi tại thời điểm user thử, không phụ thuộc hoàn toàn vào dữ liệu sống hiện tại.
- Snapshot giúp báo cáo lịch sử không bị thay đổi khi bảng gốc thay đổi.

Snapshot nên lưu dưới dạng JSONB, ví dụ:

```text
sample_spec_snapshot
wardrobe_item_spec_snapshot
trial_result_snapshot
```

---

## Rule cho Garment Spec

`garment_specs` nên được xem là gần immutable.

Không update trực tiếp các field quan trọng nếu spec đã được dùng bởi:

- `wardrobe_items`
- `digital_samples`
- `outfits`
- `sample_outfit_trials`
- `sample_trial_items`

Khi cần sửa metadata quan trọng, ưu tiên tạo spec mới.

Ví dụ:

```text
old garment_spec_id = A
new garment_spec_id = B
```

Có thể update các field ít rủi ro như `metadata.normalized_at` hoặc `updated_at`, nhưng cần tránh thay đổi category/color/material/style khiến lịch sử outfit/trial bị sai nghĩa.

---

## Giao tiếp qua contract

### Nguyên tắc

Không để module import trực tiếp use case implementation của module khác.

Không để module khác gọi trực tiếp repository của `garment`, `wardrobe` hoặc `samplelab`.

Dùng package `contract` ở cấp module theo convention hiện tại của repo.

Ví dụ:

```text
internal/modules/garment/contract
internal/modules/wardrobe/contract
internal/modules/brand/contract
internal/modules/samplelab/contract
```

---

## Contract của Garment

`garment` nên expose contract cho việc tạo, đọc, clone và snapshot spec.

Gợi ý interface:

```go
type GarmentSpecReader interface {
	GetSpec(ctx context.Context, id uuid.UUID) (*GarmentSpecView, error)
	GetSpecs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]GarmentSpecView, error)
}

type GarmentSpecWriter interface {
	CreateSpec(ctx context.Context, input CreateGarmentSpecInput) (uuid.UUID, error)
	CloneSpec(ctx context.Context, sourceID uuid.UUID, overrides GarmentSpecOverrides) (uuid.UUID, error)
	LockSpec(ctx context.Context, id uuid.UUID) error
}

type GarmentSpecSnapshotter interface {
	BuildSnapshot(ctx context.Context, id uuid.UUID) (GarmentSpecSnapshot, error)
}
```

Các struct trong contract nên là DTO nhẹ, không expose trực tiếp GORM entity nếu không cần.

---

## Contract của Wardrobe

`wardrobe` nên expose contract cho Sample Lab đọc item của user khi thử sample.

Gợi ý interface:

```go
type WardrobeItemReader interface {
	GetUserWardrobeItem(ctx context.Context, userID uuid.UUID, itemID uuid.UUID) (*WardrobeItemView, error)
	ListUserWardrobeItemsForStyling(ctx context.Context, userID uuid.UUID, filter WardrobeStylingFilter) ([]WardrobeItemView, error)
	BuildWardrobeItemSnapshot(ctx context.Context, userID uuid.UUID, itemID uuid.UUID) (WardrobeItemSnapshot, error)
}
```

`samplelab` dùng contract này để:

- Lấy danh sách item user có thể phối với sample.
- Xác minh item thật sự thuộc user.
- Tạo snapshot item tại thời điểm trial.

`samplelab` không được gọi repository của wardrobe trực tiếp.

---

## Contract của Brand

`brand` nên expose contract cho Sample Lab kiểm tra brand tồn tại và quyền brand member.

Gợi ý interface:

```go
type BrandReader interface {
	GetBrand(ctx context.Context, brandID uuid.UUID) (*BrandView, error)
}

type BrandAccessChecker interface {
	CanManageBrand(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, permission string) (bool, error)
}
```

Nếu authorization đã xử lý ở middleware hoặc route layer thì use case vẫn có thể cần contract này để double-check các thao tác quan trọng.

---

## Contract của Sample Lab

`samplelab` có thể expose contract cho Brand hoặc Campaign nếu cần lấy insight.

Gợi ý interface:

```go
type SampleInsightReader interface {
	GetSampleInsightSummary(ctx context.Context, brandID uuid.UUID, sampleID uuid.UUID) (*SampleInsightSummary, error)
}
```

Trong MVP, nếu Brand Portal gọi trực tiếp Sample Lab API thì chưa cần quá nhiều contract ngược.

---

## Luồng tạo wardrobe item

```text
User upload item
→ wardrobe nhận request
→ AI/media pipeline trích xuất metadata
→ wardrobe gọi garment.CreateSpec(metadata)
→ wardrobe tạo wardrobe_items với garment_spec_id
→ wardrobe item được tính vào capacity của user
```

Wardrobe là module điều phối use case này, nhưng metadata được tạo qua contract của `garment`.

---

## Luồng tạo digital sample

```text
Brand member tạo sample
→ samplelab nhận request
→ samplelab kiểm tra quyền brand qua brand contract
→ samplelab gọi garment.CreateSpec(metadata)
→ samplelab tạo digital_samples với garment_spec_id
→ sample sẵn sàng cho vote/trial nếu status cho phép
```

Sample Lab là module điều phối use case này, nhưng metadata được tạo qua contract của `garment`.

---

## Luồng thử phối sample với tủ đồ

```text
User chọn digital sample
→ samplelab lấy digital sample
→ samplelab lấy sample garment spec qua garment contract
→ samplelab lấy wardrobe items của user qua wardrobe contract
→ AI/rule engine phối sample với wardrobe items
→ samplelab lưu sample_outfit_trials
→ samplelab lưu sample_trial_items
→ samplelab lưu snapshot metadata sample và wardrobe items
→ optional: cộng điểm loyalty qua brand/loyalty contract nếu campaign cho phép
```

Luồng này thể hiện đúng bản chất:

- Sample Lab tương tác với tủ đồ.
- Tủ đồ vẫn thuộc wardrobe.
- Metadata chung vẫn thuộc garment.
- Brand vẫn sở hữu sample.
- Trial là entity riêng của Sample Lab.

---

## Dependency direction đề xuất

Cho phép:

```text
wardrobe → garment contract
samplelab → garment contract
samplelab → wardrobe contract
samplelab → brand contract
brand → samplelab contract nếu cần đọc insight
campaign/loyalty → samplelab contract nếu cần tính reward từ trial
```

Không cho phép:

```text
garment → wardrobe
garment → samplelab
wardrobe → samplelab
brand → wardrobe repository
samplelab → wardrobe repository
samplelab → garment repository
```

Nếu cần dữ liệu từ module khác, tạo contract phù hợp.

---

## Shared entities và ownership

Do project dùng shared entities trong `internal/shared/domain/entities`, entity Go có thể vẫn nằm chung một chỗ.

Ví dụ:

```text
internal/shared/domain/entities/garment_entities.go
internal/shared/domain/entities/wardrobe_entities.go
internal/shared/domain/entities/brand_entities.go
internal/shared/domain/entities/samplelab_entities.go
```

Tuy nhiên, ownership nghiệp vụ vẫn phải rõ:

| Entity | Module sở hữu nghiệp vụ |
|---|---|
| `GarmentSpec` | `garment` |
| `Category` | `garment` |
| `WardrobeItem` | `wardrobe` |
| `Outfit` | `wardrobe` |
| `OutfitItem` | `wardrobe` |
| `Brand` | `brand` |
| `BrandMember` | `brand` |
| `BrandCustomer` | `brand` |
| `DigitalSample` | `samplelab` |
| `SampleAsset` | `samplelab` |
| `SampleOutfitTrial` | `samplelab` |
| `SampleTrialItem` | `samplelab` |
| `SampleVote` | `samplelab` |
| `SampleFeedback` | `samplelab` |

Không để việc entity nằm chung trong `shared/domain/entities` làm mờ ownership nghiệp vụ.

---

## API surface gợi ý

### User-facing

```text
GET /api/v1/digital-samples
GET /api/v1/digital-samples/:sampleId
POST /api/v1/digital-samples/:sampleId/outfit-trials
POST /api/v1/digital-samples/:sampleId/votes
POST /api/v1/digital-samples/:sampleId/feedback
```

### Brand Portal

```text
POST /api/v1/brand-portal/brands/:brandId/digital-samples
GET /api/v1/brand-portal/brands/:brandId/digital-samples
PATCH /api/v1/brand-portal/brands/:brandId/digital-samples/:sampleId
GET /api/v1/brand-portal/brands/:brandId/digital-samples/:sampleId/insights
```

### Garment

`garment` có thể không cần public API trong MVP.

Nếu cần admin/internal API sau này, tách rõ khỏi user-facing API.

---

## Kết luận quyết định

Thiết kế cần chốt cho agent:

- Không thiết kế `digital_samples` như `wardrobe_items`.
- Không để `wardrobe` và `samplelab` cùng quản lý một bảng item.
- Tạo module `garment` để sở hữu `garment_specs` và clothing metadata dùng chung.
- `wardrobe_items` tham chiếu `garment_spec_id`.
- `digital_samples` tham chiếu `garment_spec_id`.
- `samplelab` lưu `sample_outfit_trials` và `sample_trial_items` để thể hiện tương tác giữa sample và tủ đồ.
- Trial phải lưu snapshot metadata.
- Module giao tiếp qua `contract`, không gọi trực tiếp use case/repository nội bộ của nhau.
- Shared GORM entities vẫn có thể nằm trong `internal/shared/domain/entities`, nhưng ownership nghiệp vụ phải được tài liệu hoá rõ.
