# Audit Report — Module Wardrobe

**Feature**: `013-briefres-audit`
**Phạm vi**: `internal/modules/wardrobe/application/dto/` + `internal/modules/wardrobe/presentation/handler/`
**Loại**: Audit/analysis — KHÔNG sửa code, KHÔNG đổi DB schema, KHÔNG tạo task.
**Ngày**: 2026-08-22

---

## Tổng quan

Module wardrobe có **9 response struct** (5 struct đã là BriefRes/dùng chung + 4 struct response chính) và **13 endpoint HTTP** (chưa gồm admin). Phát hiện chính:

- **Overdelivery nặng nhất**: `WardrobeItemRes` (12 fields) nhúng `FashionItemRes` đầy đủ 16 fields (gồm 3 field HSL `colorHue/colorSaturation/colorLightness` + 2 timestamp) cho **mọi context list** — dù grid chỉ cần ảnh + category + color. Đồng thời trả các field internal không cần cho display: `userId`, `isDeleted`, `taskId`.
- **Overdelivery thứ hai**: `SearchWardrobeItemRes` (endpoint public search) cũng nhúng full `FashionItemRes`.
- **Underdelivery**: `WardrobeItemRes.brandItem` (kiểu `BrandItemBriefRes`) **không bao giờ được populate** ở HTTP read path của module wardrobe (chỉ AI module fashion gán); vì vậy item nguồn gốc brand-item trong grid thiếu thông tin brand (tên/hãng/logo).
- `SearchDocumentDTO` và `WardrobeEventPayload` là struct **internal** (Elasticsearch index + event messaging) — **ngoài phạm vi** response API, chỉ ghi nhận.
- Admin endpoints (`/api/v1/admin/wardrobe-items`) theo policy spec **không đề xuất refactor** — giữ full fields cho context quản trị.
- Lưu ý ràng buộc kiến trúc: `WardrobeItemRes.FashionItem` được pipeline AI (recommendation/chat) tiêu thụ với **đầy đủ fields** — không thể co nhỏ trực tiếp struct này; cần tách `WardrobeItemBriefRes` riêng cho context display thay vì sửa `WardrobeItemRes`.

---

## Danh sách struct

| Struct | File | Số fields | Trạng thái |
|---|---|---|---|
| `CategoryBriefRes` | `wardrobe.go:18` | 3 | ✓ Đã brief |
| `FashionItemRes` | `wardrobe.go:24` | 16 | ⚠ Overdelivery (HSL + timestamps) |
| `BrandItemBriefRes` | `wardrobe.go:43` | 6 | ✓ Đã brief (cải tiến nhỏ) |
| `WardrobeItemRes` | `wardrobe.go:52` | 12 | ⚠ Overdelivery (internal fields + nhúng full FashionItemRes) |
| `SearchWardrobeItemRes` | `wardrobe.go:93` | 4 | ⚠ Overdelivery (nhúng full FashionItemRes) |
| `WardrobeStatsRes` | `wardrobe.go:142` | 2 | ✓ OK |
| `SearchDocumentDTO` | `search.go:17` | 21 | Ngoài phạm vi (ES index internal) |
| `OutfitRes` | `outfit.go:27` | 10 | ⚠ Overdelivery nhẹ (`userId`, timestamps) |
| `FashionItemBriefRes` | `outfit.go:39` | 4 | ✓ Đã brief |
| `OutfitItemRes` | `outfit.go:46` | 8 | △ Borderline (7/8 cần thiết) |
| `UnderutilizedItemInfoDTO` | `dashboard_dto.go:7` | 8 | Ngoài phạm vi HTTP (contract nội bộ dashboard) |
| `UserWardrobeSummaryStatsDTO` | `dashboard_dto.go:18` | 5 | ✓ OK (contract nội bộ) |

---

## Overdelivery

### 1. `WardrobeItemRes` — 12 fields, nhúng `FashionItemRes` (16 fields) ở mọi context

Context hiển thị (grid/list): card trang phục chỉ cần **ảnh, category, color, status, giá, trạng thái khóa**.

- **Giữ** (display): `id`, `price`, `status`, `isLocked`, `itemContext`, `fashionItem` (đổi sang `FashionItemBriefRes`), `brandItem`, `lastUsedAt`.
- **Bỏ** (internal, không dùng cho display):
  - `userId` — ID của chính user đang gọi, thừa trong mọi response.
  - `isDeleted` — flag soft-delete nội bộ, **không nên lộ ra API**.
  - `taskId` — chỉ cần trong context batch-upload (để frontend subscribe SSE); không cần ở grid/detail.
  - `createdAt` — timestamp hiếm khi hiển thị trên card; nếu cần "ngày thêm" có thể giữ ở detail riêng.
- **Vấn đề nặng nhất**: `fashionItem` nhúng full `FashionItemRes` (16 fields) cho cả list → lặp lại toàn bộ HSL + timestamps cho từng item × trang. Grid chỉ cần `FashionItemBriefRes` (đã có sẵn).

### 2. `FashionItemRes` (bản copy wardrobe, `wardrobe.go:24`) — 16 fields

- **Giữ** (detail): `id`, `category`, `imageUrl`, `color`, `colorHex`, `style`, `material`, `pattern`, `fit`, `seasonality`, `description`.
- **Bỏ**: `colorHue`, `colorSaturation`, `colorLightness` — dữ liệu AI nội bộ (phục vụ color-matching algorithm, không hiển thị). `createdAt`, `updatedAt` — không cần cho màn hình người dùng.
- ⚠ **Ràng buộc**: struct này cũng là struct `FashionItemRes` canonical ở fashion module (dùng bởi contract + AI pipeline). Nếu sửa field chung sẽ ảnh hưởng module khác → nên xử lý qua việc thêm struct display riêng (`WardrobeItemBriefRes`) hoặc chỉ bỏ field ở tầng mapper display, không sửa struct chung.

### 3. `SearchWardrobeItemRes` — 4 fields nhưng nhúng full `FashionItemRes`

- **Giữ**: `id`, `price`, `isSystem`, `fashionItem`.
- **Bỏ**: không bỏ field cấp ngoài — chỉ đổi kiểu `fashionItem` từ `*FashionItemRes` → `*FashionItemBriefRes` (grid search chỉ cần ảnh + category + màu để filter/display).

### 4. `OutfitRes` — 10 fields

- **Giữ** (list + detail): `id`, `name`, `description`, `coverImageUrl`, `status`, `items` (detail).
- **Bỏ**: `userId` (thừa), `createdAt`/`updatedAt` (list không cần; nếu cần hiển thị ngày tạo có thể giữ `createdAt` ở detail).
- List context `GET /me/outfits` hiện trả full struct nhưng `items` = nil → vẫn nặng thừa.

### 5. `OutfitItemRes` — 8 fields, borderline

7/8 fields đều cần cho canvas render (`positionX/Y`, `scale`, `layerOrder`) và display (`fashionItem`, `isAvailable`, `itemContext`). Chỉ ghi nhận: field `id` thực chất được mapper gán bằng `FashionItemID` (không phải ID của outfit-item) — naming không nhất quán, có thể gây hiểu lầm cho frontend (chi tiết ở Underdelivery).

---

## Underdelivery

### 1. `WardrobeItemRes.brandItem` không được populate ở HTTP read path (CAO)

- Field `brandItem` tồn tại nhưng **chỉ** được gán trong pipeline AI fashion (`fashion/application/mapper/recommendation.go`, `workflow_helpers.go`) — còn HTTP read path của wardrobe (`MapToWardrobeItemRes`) **không bao giờ set**.
- Hệ quả: item nguồn gốc brand-item hiển thị trong grid/chi tiết **thiếu tên hãng/logo/giá sản phẩm gốc** — frontend phải gọi thêm API brand để lấy, hoặc hiển thị thiếu thông tin.
- Nếu intent sản phẩm là hiển thị brand trên card → cần populate `brandItem` (qua contract brand) hoặc thêm trường flatten như `brandName` vào response.

### 2. `BrandItemBriefRes` thiếu `brandLogoUrl` / không dùng `BrandBriefRes` (THẤP)

- Hiện flatten `brandId` + `brandName` (duplicate tên, không có logo). Brand module đã có `BrandBriefRes` (ID, Name, Slug, Description, LogoURL, BackgroundURL).
- Nếu frontend cần logo brand trên card → thêm `brandLogoUrl *string` hoặc thay bằng nested `*brand_dto.BrandBriefRes`. Mức ưu tiên thấp vì grid hiện còn chưa có dữ liệu brand (xem #1).

### 3. `OutfitItemRes.id` gán nhầm bằng `FashionItemID` (THẤP)

- `mapper/outfit.go:42` gán `ID: item.FashionItemID`. Frontend nhận `id` của fashion item nhưng trường được đặt tên là `id` (của outfit item) → rủi ro hiểu nhầm; đồng thời `fashionItem.id` lặp lại cùng giá trị. Cần đổi tên field hoặc giữ ID thật của outfit-item.

### 4. Nested struct thiếu field hiển thị

- `FashionItemBriefRes` hiện có `id, imageUrl, colorHex, category` — đủ cho grid/outfit canvas. Nếu search grid cần filter theo `style/material/pattern/fit/seasonality` (chips lọc), cần thêm các field này vào brief → cân nhắc khi mở rộng UI (mức trung bình, chưa cần ngay).

---

## Đề xuất BriefRes

### Mới tạo (2 struct)

**1. `WardrobeItemBriefRes`** — cho context grid/list (`wardrobe.go`)
```go
type WardrobeItemBriefRes struct {
    ID          uuid.UUID              `json:"id"`
    Price       *float64               `json:"price,omitempty"`
    Status      wardrobestatus.WardrobeItemStatus `json:"status"`
    IsLocked    bool                   `json:"isLocked"`
    LastUsedAt  *time.Time             `json:"lastUsedAt,omitempty"`
    FashionItem *FashionItemBriefRes   `json:"fashionItem,omitempty"`
    BrandItem   *BrandItemBriefRes     `json:"brandItem,omitempty"`
}
```
- 7 fields, loại bỏ `userId/isDeleted/taskId/createdAt` và co `fashionItem` về brief.
- Dùng cho: `GET /me/wardrobe-items`, `POST /wardrobe-items/{id}/clone`, `POST /wardrobe-items/catalog-init`, `PATCH /wardrobe-items/{id}`, `POST /wardrobe-items/{id}/retry-analysis`.
- Riêng `POST /wardrobe-items/batch-upload` trả item đang `pending` — cần **giữ `taskId`** để frontend subscribe SSE → cân nhắc thêm `TaskID *uuid.UUID` vào brief (hoặc dùng struct riêng cho batch-upload).

**2. `OutfitBriefRes`** — cho list `GET /me/outfits` (`outfit.go`)
```go
type OutfitBriefRes struct {
    ID            uuid.UUID                 `json:"id"`
    Name          string                    `json:"name"`
    Description   *string                   `json:"description"`
    CoverImageUrl *string                   `json:"coverImageUrl"`
    Status        outfitstatus.OutfitStatus `json:"status"`
    CreatedAt     time.Time                 `json:"createdAt"` // tùy chọn, nếu UI hiển thị ngày tạo
}
```
- 5–6 fields; `GET /me/outfits` hiện trả `OutfitRes` với `items=nil` → lãng phí.

### Điều chỉnh BriefRes có sẵn (đánh giá — không mặc định đổi)

- **`FashionItemBriefRes`** (`outfit.go:39`): hợp lý cho canvas/grid. Chỉ bổ sung `style/material/pattern/fit/seasonality` nếu UI cần filter chips. **Không đổi** mặc định.
- **`BrandItemBriefRes`** (`wardrobe.go:43`): hợp lý (6 fields). Chỉ thêm `BrandLogoUrl` hoặc nested `BrandBriefRes` nếu card cần logo. **Không đổi** mặc định.
- **`CategoryBriefRes`**: OK (3 fields). Lưu ý **trùng lặp** với `CategoryBriefRes` ở fashion module (`contract_dtos.go:11`) — nên dùng chung 1 định nghĩa, nhưng là việc dọn nợ kỹ thuật, không thuộc phạm vi refactor response.

---

## Ưu tiên

| # | Đề xuất | Ưu tiên | Lý do |
|---|---|---|---|
| P1 | Tạo `WardrobeItemBriefRes` và dùng cho các endpoint list/grid | **Cao** | Grid là màn hình chính, payload lớn nhất; mỗi item hiện kéo theo 16-field `FashionItemRes` + internal fields lặp theo trang |
| P2 | `SearchWardrobeItemRes`: đổi `fashionItem` sang `FashionItemBriefRes` | **Cao** | Endpoint **public**, volume cao (tìm kiếm catalog), dữ liệu brief đã có sẵn trong ES index |
| P3 | Bỏ field internal khỏi response: `userId`, `isDeleted` (và `taskId` ngoài batch-upload) trong `WardrobeItemRes` | **Cao** | Vệ sinh dữ liệu/bảo mật — lộ flag nội bộ; đồng thời `isDeleted` gây hiểu nhầm cho FE |
| P4 | Populate `brandItem` (hoặc thêm `brandName`) trong read path wardrobe | **Cao** | Underdelivery hiện hữu: item brand thiếu thông tin brand ở grid — FE phải gọi thêm API |
| P5 | Tạo `OutfitBriefRes` cho `GET /me/outfits`; bỏ `userId` khỏi `OutfitRes` | **Trung bình** | Outfit list ít item hơn nhưng vẫn thừa field; tách list/detail theo đúng pattern |
| P6 | Bỏ HSL (`colorHue/Saturation/Lightness`) + timestamp khỏi JSON display của `FashionItemRes` | **Trung bình** | Chỉ cần thiết cho AI pipeline nội bộ; tránh đụng struct canonical → làm ở tầng display mapper |
| P7 | `BrandItemBriefRes`: thêm `brandLogoUrl` / nested `BrandBriefRes` | **Thấp** | Phụ thuộc P4; nếu grid chưa hiển thị logo thì chưa cần |
| P8 | Đổi tên/gán đúng `OutfitItemRes.id` (hiện = FashionItemID) | **Thấp** | Không tăng payload, chỉ đúng đắn về naming; ảnh hưởng contract FE |
| P9 | Dọn trùng lặp `CategoryBriefRes` / `FashionItemRes` / `FashionItemBriefRes` giữa fashion ↔ wardrobe | **Thấp** | Chất lượng code, không đổi response; ghi nhận riêng |

---

## API bị ảnh hưởng

### Endpoint hiện trả `WardrobeItemRes` (→ `WardrobeItemBriefRes` cho list)

| Endpoint | Response hiện tại | Đề xuất |
|---|---|---|
| `GET /api/v1/me/wardrobe-items` (list grid) | `PaginationResult[WardrobeItemRes]` — 12+16 fields/item, kèm userId/isDeleted/taskId + HSL | `PaginationResult[WardrobeItemBriefRes]` |
| `POST /api/v1/wardrobe-items/{id}/clone` | `[]WardrobeItemRes` | `[]WardrobeItemBriefRes` |
| `POST /api/v1/wardrobe-items/catalog-init` | `[]WardrobeItemRes` | `[]WardrobeItemBriefRes` |
| `POST /api/v1/wardrobe-items/batch-upload` | `[]WardrobeItemRes` (item pending) | `[]WardrobeItemBriefRes` **+ giữ `taskId`** (FE cần để subscribe SSE) |
| `PATCH /api/v1/wardrobe-items/{id}` | `WardrobeItemRes` | `WardrobeItemRes` (bỏ userId/isDeleted) — detail |
| `POST /api/v1/wardrobe-items/{id}/retry-analysis` | `WardrobeItemRes` | `WardrobeItemRes` (bỏ userId/isDeleted) — detail |
| `GET /api/v1/wardrobe-items/{id}` (detail) | `WardrobeItemRes` | `WardrobeItemRes` (giữ full fashion attributes, bỏ HSL/timestamps qua display mapper) |

### Endpoint trả `SearchWardrobeItemRes`

| Endpoint | Response hiện tại | Đề xuất |
|---|---|---|
| `GET /api/v1/system-catalog/wardrobe-items` (public search) | `PaginationResult[SearchWardrobeItemRes]` với `fashionItem` = full `FashionItemRes` | `fashionItem` → `*FashionItemBriefRes` (giữ `price` + `isSystem`) |

### Endpoint trả `OutfitRes`

| Endpoint | Response hiện tại | Đề xuất |
|---|---|---|
| `POST /api/v1/outfits`, `PUT /api/v1/outfits/{id}`, `GET /api/v1/outfits/{id}` | `OutfitRes` (detail) | `OutfitRes` — bỏ `userId` (+ tùy chọn giữ `createdAt` nếu UI cần) |
| `GET /api/v1/me/outfits` (list) | `PaginationResult[OutfitRes]`, `items=nil` | `PaginationResult[OutfitBriefRes]` |

### Admin / Nội bộ (không đề xuất refactor)

| Endpoint | Ghi chú |
|---|---|
| `GET /api/v1/admin/wardrobe-items`, `PUT/DELETE /admin/wardrobe-items/{id}`, admin `batch-upload` | Giữ `WardrobeItemRes` full fields cho context quản trị — theo policy spec |
| `SearchDocumentDTO`, `WardrobeEventPayload` | Struct internal (ES index / event) — ngoài phạm vi response API |

---

## Open Questions (đề xuất cho plan)

1. **Batch-upload**: có tách struct riêng chứa `taskId` hay thêm `taskId` vào `WardrobeItemBriefRes`?
2. **`WardrobeItemRes` vẫn là contract nội bộ** giữa wardrobe ↔ fashion (AI recommendation/chat tiêu thụ `FashionItem` full fields) — xác nhận không co nhỏ struct này, chỉ thêm `WardrobeItemBriefRes` cho HTTP, hay cần refactor contract?
3. **Brand thông tin**: cần `brandItem` ở grid (populate qua brand contract) hay đủ khi frontend có `fashionItem.category`?
4. **`FashionItemRes` canonical** (fashion module) có được phép chỉnh (bỏ HSL/timestamp khỏi JSON) hay chỉ làm ở tầng mapper display của wardrobe?
5. **`GET /me/outfits`** có cần `createdAt` để hiển thị "ngày tạo" hay `OutfitBriefRes` không cần field thời gian?