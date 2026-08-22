# Audit Report — Module `fashion`

**File**: `internal/modules/fashion/application/dto/` + `internal/modules/fashion/presentation/handler/`
**Giai đoạn**: Audit / plan (013-briefres-audit) — chỉ phân tích, KHÔNG sửa code.
**Chính sách backward compat**: Breaking change — bỏ field cũ, thay bằng BriefRes, thông báo qua `frontend-note.md`.

---

## 1. Tổng quan

Module `fashion` có **2 handler HTTP**: `CategoryHandler` (public + admin) và `WardrobeAIHandler` (AI recommendation + AI chat + SSE). HTTP surface của module nhỏ và gọn — phần lớn DTO trong `application/dto/` là **contract nội bộ** (gọi qua interface giữa module) hoặc **job/worker** (hàng đợi, SSE), không phải response HTTP trực tiếp.

Phát hiện chính:

- **Overdelivery thật sự nằm ở lớp contract**: `FashionItemRes` (17 fields) là contract nội bộ nhưng bị nhúng vào payload HTTP của module khác qua `WardrobeItemRes.FashionItem` (closet) và `BrandItemRes.FashionItem` (brand items). Các field HSL (`colorHue/colorSaturation/colorLightness`) và `createdAt/updatedAt` chỉ phục vụ thuật toán AI (scoring/HSL ranking, chat prompt) — frontend không cần nhưng vẫn bị serialize ra.
- **Overdelivery trực tiếp (HTTP của fashion)**: `RecommendedFashionItemRes` (8 fields) — `material/pattern/fit` không cần cho card phối đồ; `SSEEventPayload.data` (batch upload) chứa full `WardrobeItemRes`.
- **Underdelivery**: module `fashion` gần như không có. Các response đều đã nhúng `CategoryBriefRes`/`BrandItemBriefRes`. Điểm đáng lưu ý duy nhất là `FashionItemBriefRes` thiếu field `color` (tên màu) nếu frontend cần hiển thị.
- **Dead code**: `PendingTransferRes` được khai báo nhưng không dùng ở bất kỳ đâu (chỉ xuất hiện trong file DTO).
- Các struct BriefRes sẵn có (`CategoryBriefRes`, `FashionItemBriefRes`) đang đúng chuẩn — không mặc định thay đổi.

---

## 2. Danh sách struct

### 2.1 Response HTTP trực tiếp (qua handler của fashion)

| # | Struct | Fields | Endpoint | Ghi chú |
|---|--------|--------|----------|---------|
| 1 | `CategoryAPIRes` | 4 (ID, Name, Slug, SortOrder) | `/api/v1/categories`, `/api/v1/admin/categories[...]` | OK |
| 2 | `RecommendedOutfitRes` | 5 (Title, Explanation, Items, IsFallback, RemainingQuota) | `POST /api/v1/ai/outfit-recommendations` | OK |
| 3 | `RecommendedItemGroup` | 3 (Role, Primary, Alternatives) | nested trong #2 | OK |
| 4 | `RecommendedItemRes` | 4 (ID, ItemContext, BrandItem, FashionItem) | nested trong #2 | OK |
| 5 | `RecommendedFashionItemRes` | 8 (ID, Category, ImageUrl, Color, ColorHex, Style, Material, Pattern, Fit) | nested trong #4 | **Overdelivery** |
| 6 | `ChatSessionRes` | 6 (ID, Title, ContextSummary, IsArchived, CreatedAt, UpdatedAt) | `POST/GET/PATCH /api/v1/ai/chat/sessions[...]` | Nhẹ |
| 7 | `ChatMessageRes` | 4 (ID, Sender, Content, CreatedAt) | `GET /api/v1/ai/chat/sessions/{id}/messages` | OK |

### 2.2 Contract nội bộ / worker (KHÔNG phải response HTTP của fashion, nhưng ảnh hưởng payload của module khác)

| # | Struct | Fields | Mục đích |
|---|--------|--------|----------|
| 8 | `FashionItemRes` | 17 (ID, Category, ImageUrl, ImagePublicID(`json:"-"`), Color, ColorHex, ColorHue, ColorSaturation, ColorLightness, Style, Material, Pattern, Fit, Seasonality, Description, CreatedAt, UpdatedAt) | Contract `GetFashionItem`/`ListFashionItems`; nhúng vào HTTP qua wardrobe/brand |
| 9 | `FashionItemBriefRes` | 4 (ID, ImageUrl, ColorHex, Category) | Contract `ListFashionItemsBrief` → outfit items (wardrobe) |
| 10 | `CategoryBriefRes` | 3 (ID, Name, Slug) | Nested trong FashionItemRes |
| 11 | `CategoryRes` | 4 (ID, Name, Slug, SortOrder) | Contract category (brand dùng) |
| 12 | `FashionProcessingInfoDTO` | 4 | Internal (stale processing) |
| 13 | `DashboardCategoryDTO` | 2 (ID, Name) | Internal (dashboard) |
| 14 | `DashboardFashionItemCategoryDTO` | 2 (FashionItemID, CategoryID) | Internal (dashboard, batch) |
| 15 | `FashionAnalyzeJobDTO` | 12 | Job hàng đợi (không ra HTTP) |
| 16 | `SSEEventPayload` | 6 (ItemID, Status, Total, Index, Data, Error) | SSE batch upload — `Data` là `any` |
| 17 | `PendingTransferRes` | 3 (PostItemID, Item, SellerName) | **Dead code** — không dùng |
| 18 | `FashionMetadataResult` | 11 | Parse kết quả AI (internal) |
| 19 | `AICategoryRef` | 2 (Name, Slug) | Internal |
| 20 | `ParsedIntent` | 9 | Parse intent người dùng (internal) |

---

## 3. Overdelivery

### 3.1 `RecommendedFashionItemRes` (8 fields) — HTTP trực tiếp

Context hiển thị: card món đồ trong kết quả phối đồ (primary + 2 alternatives × 5-8 vai trò). Frontend cần: ảnh, màu (hex/name), danh mục, phong cách, id.

- **Giữ**: `ID`, `Category` (CategoryBriefRes), `ImageUrl`, `ColorHex`, `Color`, `Style`
- **Bỏ**: `Material`, `Pattern`, `Fit` (không hiển thị trong card phối đồ; nếu người dùng cần chi tiết, có thể xem qua trang item — không thuộc context này)
- Lý do giữ `Style`: thường được hiển thị để giải thích lý do AI chọn món đó (mặc dù có thể tùy chọn).

### 3.2 `SSEEventPayload.Data` — `any` chứa full `WardrobeItemRes`

Khi status `completed`, worker truyền `data = WardrobeItemRes` (đầy đủ) từ wardrobe contract — trong đó `FashionItem` là full `FashionItemRes` (kèm HSL, createdAt, updatedAt). Context: frontend append item vào danh sách tủ đồ.

- **Giữ**: thông tin item tối thiểu để append + hiển thị (id, ảnh, màu, danh mục, price, status)
- **Bỏ**: `colorHue/colorSaturation/colorLightness`, `createdAt/updatedAt` của fashion item, `imagePublicId`
- Lưu ý: `Data` là `any` nên không thể ép qua struct — cần đổi tại nguồn phát (worker + wardrobe contract) → cross-module.

### 3.3 `FashionItemRes` (17 fields) — nhúng qua module khác

Không phải response HTTP của fashion, nhưng bị serialize vào HTTP của wardrobe (closet list/detail) và brand (brand item list/detail).

- **Detail context (giữ)**: `ID`, `Category`, `ImageUrl`, `Color`, `ColorHex`, `Style`, `Material`, `Pattern`, `Fit`, `Seasonality`, `Description`
- **List context (nên bỏ)**: `ColorHue`, `ColorSaturation`, `ColorLightness` (chỉ dùng server-side cho HSL ranking & chat prompt), `CreatedAt`, `UpdatedAt`
- Lý do KHÔNG xóa field khỏi struct này: các field HSL được thuật toán AI (fallback_hsl, scoring) và chat prompt sử dụng trực tiếp trên struct này. → Không trim struct, mà **đề xuất consumer list dùng `ListFashionItemsBrief`** (đã có sẵn `FashionItemBriefRes`).

### 3.4 `ChatSessionRes` (6 fields) — nhẹ, tùy chọn

`ContextSummary` (chuỗi tóm tắt hội thoại, có thể vài trăm ký tự × N session) hơi nặng cho list. Tuy nhiên không có endpoint "chi tiết session" riêng — frontend có thể cần summary để render preview trong danh sách. **Đánh giá: giữ nguyên**, chỉ đề xuất nếu summary trở nên lớn (xem §5.2).

### 3.5 Internal structs — KHÔNG refactor (ghi nhận)

`FashionAnalyzeJobDTO` (12), `FashionMetadataResult` (11), `ParsedIntent` (9): internal, không ra HTTP — không đề xuất.

---

## 4. Underdelivery

Module `fashion` **không có underdelivery nghiêm trọng**. Rà soát:

- `RecommendedFashionItemRes.Category` — đã là `CategoryBriefRes` (id + name + slug) ✓
- `RecommendedItemRes.BrandItem` — đã là `BrandItemBriefRes` (id, brandId, brandName, itemType, name, price) ✓ — frontend có brandName ngay, không cần gọi thêm.
- `RecommendedItemRes.ID` — là wardrobe item ID; fashion item ID nằm trong `FashionItem.ID` (nested). Cần ghi rõ semantics trong API docs nhưng không thiếu dữ liệu.
- `DashboardFashionItemCategoryDTO` trả `categoryId` bare — nhưng là contract internal, dashboard batch `GetCategoriesByIDs` sau đó. Không ảnh hưởng frontend.
- `SSEEventPayload.ItemID` — wardrobe item ID; khi `completed`, `Data` chứa đủ thông tin item. OK.
- Chat không trả nested item (text chat) — OK.

**Điểm nhỏ (tùy chọn)**: `FashionItemBriefRes` (cả bản fashion lẫn wardrobe) chỉ có `ColorHex`, thiếu `Color` (tên màu). Nếu outfit card cần hiển thị tên màu, cần thêm field `Color *string`. Không mặc định thêm.

---

## 5. Đề xuất BriefRes

### 5.1 `RecommendedFashionItemBriefRes` — mới

Thay thế `RecommendedFashionItemRes` trong `RecommendedItemRes.FashionItem`.

```go
type RecommendedFashionItemBriefRes struct {
    ID       uuid.UUID         `json:"id"`
    Category *CategoryBriefRes `json:"category,omitempty"`
    ImageUrl string            `json:"imageUrl,omitempty"`
    Color    *string           `json:"color,omitempty"`
    ColorHex *string           `json:"colorHex,omitempty"`
    Style    *string           `json:"style,omitempty"`
}
```

Lý do đặt tên theo convention `<Entity>BriefRes` (entity = RecommendedFashionItem). Giảm từ 8 → 6 fields, cắt `material/pattern/fit`.

### 5.2 `ChatSessionBriefRes` — mới (chỉ khi ContextSummary nặng)

Dùng cho `GET /api/v1/ai/chat/sessions` (list).

```go
type ChatSessionBriefRes struct {
    ID         uuid.UUID `json:"id"`
    Title      string    `json:"title"`
    IsArchived bool      `json:"isArchived"`
    UpdatedAt  time.Time `json:"updatedAt"`
}
```

**Không mặc định tạo** — chỉ thực hiện nếu `contextSummary` tăng kích thước. Ghi nhận là open question (xem §7).

### 5.3 Đánh giá các BriefRes sẵn có (FR-009)

| Struct | Đánh giá | Đề xuất |
|--------|----------|---------|
| `CategoryBriefRes` (fashion) | Đúng chuẩn (3 fields), trùng struct cùng tên ở wardrobe | Giữ nguyên. Cân nhắc thống nhất dùng chung 1 struct 2 module — out of scope |
| `FashionItemBriefRes` (fashion) | Đúng chuẩn (4 fields) | Giữ nguyên. Optional: thêm `Color` (tên màu) nếu outfit card cần — ưu tiên thấp |
| `BrandItemBriefRes` (wardrobe, dùng trong recommendation) | Đầy đủ (6 fields, có brandName) | Giữ nguyên |

---

## 6. Ưu tiên

| Ưu tiên | Đề xuất | Lý do |
|---------|---------|-------|
| **Cao** | Trim `RecommendedFashionItemRes` → `RecommendedFashionItemBriefRes` (§5.1) | Endpoint AI gọi thường xuyên; mỗi outfit lặp item nhiều lần (primary + 2 alternatives × 5-8 roles) → giảm tải rõ rệt; phạm vi 1 struct + mapper, ít rủi ro |
| **Cao** | SSE `completed` dùng brief data thay full `WardrobeItemRes` (§3.2) | Batch upload phát nhiều event SSE, mỗi event chứa full wardrobe item (kèm HSL/createdAt/updatedAt) → payload phình; frontend chỉ cần append item |
| **Trung bình** | List context (wardrobe/brand) nhúng `FashionItemRes` đầy đủ → dùng `FashionItemBriefRes`/`ListFashionItemsBrief` (§3.3) | Tác động payload lớn nhất hệ thống nhưng nằm ở module khác (cross-module, cần phối hợp) — ghi nhận cho wardrobe/brand audit |
| **Trung bình** | `ChatSessionBriefRes` cho GET sessions list (§5.2) | Giảm tải list; nhưng ContextSummary hiện có thể cần để preview → cần dev quyết định |
| **Thấp** | Thêm `Color` vào `FashionItemBriefRes` | Chỉ khi outfit card cần hiển thị tên màu; không mặc định thêm |
| **Thấp** | Xoá `PendingTransferRes` (dead code) | Không phải response API, chỉ dọn dẹp |

---

## 7. Open Questions

1. **SSE completed**: có nên đổi nguồn phát để `data` là struct brief (không dùng `any`) không? Phạm vi lan sang wardrobe contract (`GetItemsByIDs`).
2. **ChatSessionRes**: `ContextSummary` có cần giữ trong list không? Nếu có, không tạo `ChatSessionBriefRes`.
3. **FashionItemRes**: việc nhúng full struct vào list của wardrobe/brand sẽ được xử lý ở module nào (đây là fashion audit — ghi nhận để chuyển cho audit wardrobe/brand)?

---

## 8. API bị ảnh hưởng

| Endpoint | Response hiện tại | Response đề xuất | Ưu tiên |
|----------|-------------------|------------------|---------|
| `POST /api/v1/ai/outfit-recommendations` | `RecommendedOutfitRes.items[].primary/alternatives[].fashionItem` = `RecommendedFashionItemRes` (8 fields) | `RecommendedFashionItemBriefRes` (6 fields, bỏ material/pattern/fit) | Cao |
| SSE batch upload wardrobe items (SSEEventPayload, `sseManager.Publish`) | `data` (status=completed) = full `WardrobeItemRes` | `data` = brief item (id, image, colorHex, category, price, status) | Cao |
| `GET /api/v1/ai/chat/sessions` | `[]ChatSessionRes` (6 fields) | `[]ChatSessionBriefRes` (4 fields) — nếu chốt §5.2 | Trung bình |
| (Cross-module) wardrobe items list/detail — `WardrobeItemRes.FashionItem` | full `FashionItemRes` | `FashionItemBriefRes` (list context) | Trung bình |
| (Cross-module) brand items list/detail — `BrandItemRes.FashionItem` | full `FashionItemRes` | `FashionItemBriefRes` (list context) | Trung bình |
| `GET /api/v1/categories`, `GET/POST/PUT /api/v1/admin/categories[...]` | `CategoryAPIRes` | Không đổi | — |
| `POST/GET/PATCH /api/v1/ai/chat/sessions[...]`, `GET .../messages` | `ChatSessionRes`, `ChatMessageRes` | Không đổi (trừ §5.2) | — |

---

## 9. Kết luận

Module `fashion` có HTTP surface nhỏ nên mức độ overdelivery trực tiếp thấp — chỉ 2 điểm đáng xử lý ưu tiên cao: `RecommendedFashionItemBriefRes` (outfit recommendation) và SSE completed payload. Vấn đề payload lớn nhất (`FashionItemRes` nhúng qua wardrobe/brand) là cross-module và cần được chuyển sang audit của wardrobe/brand. Module không có underdelivery đáng kể; các BriefRes sẵn có đang đạt chuẩn, không cần thay đổi mặc định.