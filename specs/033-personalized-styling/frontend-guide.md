# Hướng dẫn tích hợp Frontend - Cá nhân hoá phối đồ và gợi ý bổ sung

**Feature**: `033-personalized-styling` | **Task**: T084 | **Date**: 2026-10-27

Tài liệu này là hợp đồng tích hợp cho **hai client**:

- `[WEB]` = repo `smart-wardrobe-fe` (Next.js 16 App Router, TanStack Query, Zustand, Axios, shadcn/ui)
- `[MOBILE]` = repo `smart-wardrobe-mobile` (Flutter, Riverpod, go_router, Dio)

Đường dẫn backend ghi theo repo backend (`smart-wardrobe-be`) bằng đường dẫn tương đối. Đường dẫn client
được ghi theo repo tương ứng, có tiền tố `[WEB]` hoặc `[MOBILE]` để không nhầm lẫn.

Nguồn đối chiếu: `specs/033-personalized-styling/contracts/personalization-api.md`,
`specs/033-personalized-styling/data-model.md`, `specs/033-personalized-styling/quickstart.md`, và mã nguồn
backend đã triển khai (danh sách file ở §15).

Tài liệu **không chứa code triển khai**. Các đoạn JSON/`interface`/`class` chỉ mô tả hình dạng dữ liệu.

---

## 1. Phạm vi thay đổi API

Toàn bộ thay đổi là **additive**. Không có endpoint bị xoá, không có field cũ bị đổi tên/đổi kiểu.

### 1.1 Bảng tổng hợp

| # | Method | Path | Trạng thái | Ghi chú |
|---|---|---|---|---|
| 1 | GET | `/api/v1/me/style-profile` | **NEW** | Đọc hồ sơ gu; trả mặc định nếu chưa có row |
| 2 | PUT | `/api/v1/me/style-profile` | **NEW** | Partial update hồ sơ gu |
| 3 | DELETE | `/api/v1/me/style-profile` | **NEW** | Xoá hồ sơ (idempotent) |
| 4 | GET | `/api/v1/me/style-profile/suggestions` | **NEW** | Gợi ý gu từ tủ đồ (cold start), read-only |
| 5 | POST | `/api/v1/ai/outfit-recommendations` | **CHANGED** | Response thêm `recommendationId`, `suggestions[]` |
| 6 | POST | `/api/v1/ai/outfit-recommendations/feedback` | **NEW** | Feedback `like \| dislike \| saved` |
| 7 | POST | `/api/v1/ai/outfit-completions` | **NEW** | Hoàn thiện bộ đồ đã sở hữu; không gọi LLM, không trừ quota |
| 8 | POST | `/api/v1/outfits` | **CHANGED** | Request item thêm `itemContext` (optional); clone `system_catalog` |
| 9 | PUT | `/api/v1/outfits/:id` | **CHANGED** | Cùng quy tắc `itemContext` + clone như POST |
| 10 | GET | `/api/v1/subscriptions/me` | **CHANGED** | Response thêm `personalizationEnabled` |
| 11 | GET | `/api/v1/subscriptions/plans` | **CHANGED** | Plan brief thêm `personalizationEnabled` |
| 12 | GET | `/api/v1/admin/subscriptions/plans` | **CHANGED** | Admin plan DTO thêm `personalizationEnabled` |
| 13 | POST | `/api/v1/admin/subscriptions/plans` | **CHANGED** | Request nhận `personalizationEnabled` (optional, mặc định `false`) |
| 14 | PUT | `/api/v1/admin/subscriptions/plans/:id` | **CHANGED** | Request nhận `personalizationEnabled` (optional, bỏ trống = giữ nguyên) |
| 15 | — | RabbitMQ topic `wardrobe.outfit.saved` | **NEW (backend-only)** | Client **không** có hành động nào; xem §10b |

Tổng: **14 endpoint HTTP + 1 event**. Trong đó **6 endpoint HTTP mới**, **8 endpoint HTTP thay đổi**,
**0 endpoint bị xoá**.

### 1.2 Chi tiết "trước → sau" của các endpoint CHANGED

| # | Endpoint | Trước | Sau |
|---|---|---|---|
| 5 | `POST /api/v1/ai/outfit-recommendations` | `{title, explanation, items[], isFallback, remainingQuota}` | thêm `recommendationId` (luôn có) và `suggestions[]` (`omitempty`, vắng mặt khi rỗng) |
| 8 | `POST /api/v1/outfits` | item: `{fashionItemId, positionX, positionY, scale, layerOrder}` | item nhận thêm `itemContext` (optional). `system_catalog` được clone thành `wardrobe_items` trong cùng transaction; `OutfitItemRes.itemContext` trả về luôn là `user_wardrobe` hoặc `brand_item` |
| 9 | `PUT /api/v1/outfits/:id` | như trên | như trên + chỉ item **mới thêm** mới cập nhật `last_used_at` và phát signal học gu |
| 10 | `GET /api/v1/subscriptions/me` | 9 field | thêm `personalizationEnabled: boolean` |
| 11 | `GET /api/v1/subscriptions/plans` | plan brief | thêm `personalizationEnabled: boolean` |
| 12 | `GET /api/v1/admin/subscriptions/plans` | admin plan DTO | thêm `personalizationEnabled: boolean` |
| 13 | `POST /api/v1/admin/subscriptions/plans` | — | nhận `personalizationEnabled?: boolean`; bỏ trống → `false` |
| 14 | `PUT /api/v1/admin/subscriptions/plans/:id` | — | nhận `personalizationEnabled?: boolean`; bỏ trống → giữ nguyên |

### 1.3 Quy ước envelope (không đổi)

Thành công — `shared/presentation/response.go`:

```json
{ "message": "Lấy hồ sơ phong cách thành công", "data": { } }
```

- `message` và `data` đều `omitempty` → **DELETE** trả về chỉ `{ "message": "..." }`, **không có key
  `data`** (vì `data` là `nil`). Client không được giả định `data` luôn tồn tại.
- `POST /api/v1/outfits` trả **201**, các endpoint còn lại của 033 trả **200**.

Lỗi — `shared/application/constants/apperror/error.go`:

```json
{
  "status": 403,
  "title": "Không có quyền truy cập",
  "message": "Cá nhân hoá phong cách là quyền lợi của gói trả phí. Vui lòng nâng cấp gói để sử dụng tính năng này.",
  "errors": [{ "field": "styles", "message": "..." }]
}
```

- `title` là **tiếng Việt** theo constructor của `apperror` (`NewForbidden` → "Không có quyền truy
  cập", `NewBadRequest` → "Thao tác không thành công", `NewNotFound` → "Không tìm thấy dữ liệu",
  `NewInternalError` → "Lỗi hệ thống"). Đừng so sánh với tên HTTP tiếng Anh.
- **Không có field `code`.** Client phân nhánh theo **HTTP status + `message`** (contract §7). Đừng dò
  chuỗi `PERSONALIZATION_NOT_AVAILABLE` trong response — đó là tên constructor nội bộ.
- `errors[]` chỉ xuất hiện với lỗi binding/validate (400).
- Ở môi trường **không phải production**, envelope lỗi còn có thể kèm `debugMessage` và
  `stackTrace` (`shared/presentation/middleware/error_handler.go`). Hai field này bị bỏ ở production;
  client phải bỏ qua, không hiển thị và không phụ thuộc.

Xác thực: các endpoint 033 nằm trong nhóm private với `RolesAuthorize(roleslug.User)`
(`roleslug.User = "user"`, `roleslug.Admin = "admin"`).

---

## 2. `GET /api/v1/me/style-profile` — NEW

Quyền: role `user`. Không có row → trả **giá trị mặc định, không tạo row**.

**Request**: không có body, không có query.

**Response 200** — `data` = `StyleProfileRes`
(`services/commerce/internal/modules/fashion/application/dto/style_profile.go`):

```json
{
  "styles": ["minimalist", "elegant"],
  "excludedStyles": ["streetwear"],
  "enabled": true,
  "personalizationAvailable": true,
  "hasEmbedding": true,
  "maxStyles": 4,
  "updatedAt": "2026-10-27T09:00:00Z"
}
```

| Field | Kiểu | Bắt buộc | Ghi chú |
|---|---|---|---|
| `styles` | `string[]` | luôn có | Mảng rỗng nghĩa là "không có gu ưu tiên". Chưa có row → `[]`, không phải `null` |
| `excludedStyles` | `string[]` | luôn có | Gu muốn tránh; vẫn được áp dụng khi `styles` rỗng |
| `enabled` | `boolean` | luôn có | Chưa có row → `true` |
| `personalizationAvailable` | `boolean` | luôn có | Lấy từ `UserSubscriptionContractDTO.personalizationEnabled` của plan **hiện tại**. Hết quyền lợi vẫn đọc được, field này `false` |
| `hasEmbedding` | `boolean` | luôn có | `style_embedding` có dữ liệu (vector khác rỗng). Fail-soft: provider lỗi → `false` nhưng vẫn 200 |
| `maxStyles` | `number` | luôn có | Giới hạn mỗi danh sách, từ `personalization.max_styles` (mặc định 4) |
| `updatedAt` | `string \| null` | luôn có | `null` khi user chưa từng lưu hồ sơ. Client phải chấp nhận `null` |

**Trạng thái**: `200`, `401`, `403` (sai role), `500`.

---

## 3. `PUT /api/v1/me/style-profile` — NEW

Partial update: field bỏ trống = giữ nguyên; mảng rỗng = xoá danh sách.

**Request body** — `UpdateStyleProfileReq` (mọi field là con trỏ để phân biệt "không gửi" với "gửi rỗng"):

```json
{ "styles": ["minimalist", "elegant"], "excludedStyles": ["streetwear"], "enabled": true }
```

| Field | Kiểu | Bắt buộc | Ghi chú |
|---|---|---|---|
| `styles` | `string[]` | không | Trim, bỏ phần tử rỗng, dedupe không phân biệt hoa thường, mỗi giá trị ≤ 50 ký tự (NFC), tối đa `maxStyles`. Gửi `[]` = xoá danh sách |
| `excludedStyles` | `string[]` | không | Cùng quy tắc với `styles` |
| `enabled` | `boolean` | không | Bật/tắt áp dụng; tắt **không** xoá dữ liệu |

Không có validate theo từ vựng: giá trị lạ vẫn hợp lệ và đi qua nhánh embedding/text-level.

**Response 200**: giống hệt §2 (đọc lại từ DB sau khi upsert).

**Trạng thái**:

| Status | Khi nào |
|---|---|
| `200` | Lưu thành công (kể cả khi sinh embedding thất bại → `hasEmbedding: false`) |
| `400` | Vượt `maxStyles` (`STYLE_LIMIT_EXCEEDED`) hoặc một giá trị > 50 ký tự (`STYLE_VALUE_INVALID`) hoặc lỗi binding |
| `401` | Thiếu/hết hạn token |
| `403` | `PERSONALIZATION_NOT_AVAILABLE` (plan không có quyền) hoặc `PERSONALIZATION_DISABLED` (kill switch vận hành) |
| `500` | Lỗi hệ thống |

Thứ tự kiểm tra phía backend: kill switch → quyền lợi → tải row → chuẩn hoá → embedding → upsert. Vì vậy
user không có quyền lợi nhận 403 dù request sai định dạng danh sách.

---

## 4. `DELETE /api/v1/me/style-profile` — NEW

Idempotent: xoá khi chưa có row vẫn `200`.

**Request**: không body.

**Response 200**:

```json
{ "message": "Xoá hồ sơ phong cách thành công" }
```

| Field | Kiểu | Bắt buộc | Ghi chú |
|---|---|---|---|
| `message` | `string` | luôn có | Envelope message |
| `data` | — | **không có** | `data = nil` + `omitempty` → key vắng mặt. Client dùng `res.data?.data` chứ không được `res.data.data!` |

**Trạng thái**: `200`, `401`, `403` (`PERSONALIZATION_NOT_AVAILABLE` / `PERSONALIZATION_DISABLED`), `500`.

Lưu ý nghiệp vụ: user đã tự tắt `enabled = false` **vẫn** được xoá (xoá dữ liệu là quyền của chủ sở hữu).
Chỉ kill switch vận hành và việc thiếu quyền lợi mới chặn.

---

## 5. `GET /api/v1/me/style-profile/suggestions` — NEW

Gợi ý gu từ tủ đồ hiện có (cold start). **Read-only**: không lưu gì, không tự áp dụng. Kill switch
`personalization.enabled = false` **không** chặn endpoint này, nhưng thiếu quyền lợi thì vẫn 403.

**Response 200** — `data` = `StyleSuggestionRes`:

```json
{
  "styles": [
    { "value": "minimalist", "itemCount": 12, "ratio": 0.4 },
    { "value": "elegant", "itemCount": 7, "ratio": 0.23 }
  ],
  "sampleSize": 30
}
```

| Field | Kiểu | Bắt buộc | Ghi chú |
|---|---|---|---|
| `styles` | `array` | luôn có | Chưa đủ dữ liệu → `[]` (không phải `null`), không phải lỗi |
| `styles[].value` | `string` | luôn có | Nhãn phong cách đã chuẩn hoá |
| `styles[].itemCount` | `number` | luôn có | Số món đang hoạt động mang phong cách này |
| `styles[].ratio` | `number` | luôn có | `itemCount / sampleSize`, làm tròn 2 chữ số |
| `sampleSize` | `number` | luôn có | Số active item có `style` không rỗng. `0` = chưa đủ dữ liệu. Lưu ý: code còn dedupe theo `fashion_item_id` và cap theo `maxWardrobeItems` của gói, nên con số có thể nhỏ hơn "số row active" theo nghĩa đen — xem `deferred.md` D-07 |

Thông báo "Chưa đủ dữ liệu để gợi ý phong cách..." nằm ở **`message` của envelope**, không nằm trong
`data`. Client nên hiển thị `message` khi `sampleSize === 0`.

**Trạng thái**: `200`, `401`, `403` (`PERSONALIZATION_NOT_AVAILABLE`), `500`.

---

## 6. `POST /api/v1/ai/outfit-recommendations` — CHANGED

Request **không đổi**. Chỉ response mở rộng.

**Request body** — `RecommendOutfitReq`:

| Field | Kiểu | Bắt buộc | Ghi chú |
|---|---|---|---|
| `occasion` | `string` | không | Tự do; gợi ý `casual`, `work`, `date`, `party`, `sport` |
| `styleTarget` | `string` | không | Tự do; gợi ý `minimalist`, `vintage`, `streetwear`, `preppy`, `sporty`, `elegant`. **Thắng** gu đã lưu khi có |
| `season` | `string` | không | Enum: `spring \| summer \| autumn \| winter \| all` |
| `weather` | `string` | không | Tự do; gợi ý `hot`, `cold`, `warm`, `cool`, `rainy` |
| `details` | `string` | không | Free text, ≤ 1000 ký tự NFC |
| `colorTone` | `string` | không | Tự do; gợi ý `light`, `dark`, `pastel`, `earthy`, `neon` |
| `includeBrandItems` | `boolean` | không | Cho phép phối đồ brand (tối đa 30%) |

**Response 200** — `data` = `RecommendedOutfitRes`:

```json
{
  "recommendationId": "5b0c9f54-7d61-4f7a-9a4e-9f1f7f4f3a21",
  "title": "Bộ đồ công sở tối giản",
  "explanation": "...",
  "items": [
    {
      "role": "top",
      "primary": {
        "id": "uuid-wardrobe-item",
        "itemContext": "user_wardrobe",
        "fashionItem": {
          "id": "uuid-fashion-item",
          "category": { "id": "uuid", "name": "Áo", "slug": "ao" },
          "imageUrl": "https://...",
          "color": "Trắng",
          "colorHex": "#ffffff",
          "style": "Tối giản"
        }
      },
      "alternatives": []
    }
  ],
  "isFallback": false,
  "remainingQuota": 4,
  "suggestions": [
    {
      "role": "footwear",
      "reason": "Tủ đồ chưa có giày phù hợp để hoàn thiện bộ này",
      "fashionItem": {
        "id": "uuid",
        "category": { "id": "uuid", "name": "Giày", "slug": "giay" },
        "imageUrl": "https://...",
        "color": "Trắng",
        "colorHex": "#ffffff",
        "style": "Tối giản"
      }
    }
  ]
}
```

| Đường dẫn field | Kiểu | Bắt buộc | Ghi chú |
|---|---|---|---|
| `recommendationId` | `string` (uuid) | **luôn có** | **MỚI**. Khóa để gọi feedback. Có từ US1, kể cả khi `isFallback = true` |
| `title` | `string` | luôn có | Không đổi |
| `explanation` | `string` | luôn có | Không đổi |
| `items` | `array` | luôn có | Không đổi. Món gợi ý bổ sung **không** nằm trong đây |
| `isFallback` | `boolean` | luôn có | Không đổi |
| `remainingQuota` | `number` | luôn có | Không đổi. Suggestions **không** trừ quota |
| `suggestions` | `array` | **có thể vắng mặt** | **MỚI**. `omitempty`: vắng key khi rỗng. Tối đa 1 món/role, 3 món/response |
| `items[].role` | `string` | luôn có | Enum role, xem §11.1 |
| `items[].primary` | `object` | luôn có | |
| `items[].alternatives` | `array` | luôn có | Có thể `[]` |
| `items[].primary.id` | `string` | luôn có | ID **wardrobe item** (không phải fashion item) |
| `items[].primary.itemContext` | `string` | có thể vắng | Enum lưu trữ, xem §11.2 |
| `items[].primary.fashionItem` | `object` | có thể vắng | Vắng với ghost/brand item không có fashion item |
| `items[].primary.brandItem` | `object` | có thể vắng | Chỉ có khi `itemContext = brand_item` |
| `items[].primary.fashionItem.id` | `string` | luôn có | **Đây** là ID dùng cho `POST /outfits` |
| `items[].primary.fashionItem.category` | `object` | có thể vắng | `{id, name, slug}` |
| `items[].primary.fashionItem.imageUrl` | `string` | có thể vắng | |
| `items[].primary.fashionItem.color` | `string` | có thể vắng | |
| `items[].primary.fashionItem.colorHex` | `string` | có thể vắng | |
| `items[].primary.fashionItem.style` | `string` | có thể vắng | |
| `suggestions[].role` | `string` | luôn có | Role còn thiếu/chưa phù hợp. Enum như §11.1 |
| `suggestions[].reason` | `string` | luôn có | Câu giải thích tiếng Việt để hiển thị |
| `suggestions[].fashionItem` | `object` | luôn có | Cùng cấu trúc `fashionItem` ở trên (`id, category, imageUrl, color, colorHex, style`) |

**Trạng thái**:

| Status | Khi nào |
|---|---|
| `200` | Thành công (kể cả `isFallback = true`) |
| `400` | Hết lượt AI trong ngày (`Bạn đã dùng hết lượt tạo trang phục bằng AI...`), tủ đồ dưới 5 món, hoặc không ghép được bộ (`Chưa ghép được bộ hoàn chỉnh...`), hoặc lỗi binding |
| `401` | Thiếu/hết hạn token |
| `403` | Sai role |
| `500` | Lỗi hệ thống |

---

## 7. `POST /api/v1/ai/outfit-recommendations/feedback` — NEW

**Request body** — `SubmitRecommendationFeedbackReq`:

```json
{ "recommendationId": "5b0c9f54-7d61-4f7a-9a4e-9f1f7f4f3a21", "action": "like" }
```

| Field | Kiểu | Bắt buộc | Ghi chú |
|---|---|---|---|
| `recommendationId` | `string` (uuid) | **có** | Lấy từ response §6. Client **không** gửi danh sách món hay style: server đọc từ snapshot |
| `action` | `string` | **có** | Enum `like \| dislike \| saved` (§11.3). Binding là `oneof` **phân biệt hoa thường** → luôn gửi chữ thường |

**Response 200** — `data` = `SubmitRecommendationFeedbackRes`:

| Field | Kiểu | Bắt buộc | Ghi chú |
|---|---|---|---|
| `accepted` | `boolean` | luôn có | `true` khi đã ghi nhận (kể cả lần gửi lặp) |

```json
{ "message": "Ghi nhận phản hồi gợi ý thành công", "data": { "accepted": true } }
```

Idempotent theo `(user, recommendationId, action)`: gửi lại cùng action **không** tạo signal thứ hai.
`like` và `dislike` trên cùng một `recommendationId` là **hai** signal khác nhau (khác `action`) — UI nên
khoá/mở khoá nút theo lựa chọn hiện tại, không dựa vào `accepted`.

**Trạng thái**:

| Status | Khi nào |
|---|---|
| `200` | Ghi nhận thành công, kể cả snapshot của user **không** có quyền cá nhân hoá (endpoint không gate quyền lợi) |
| `400` | `action` ngoài enum (lỗi binding) hoặc request thiếu field |
| `401` | Thiếu/hết hạn token |
| `403` | `FEEDBACK_FORBIDDEN` — snapshot thuộc user khác |
| `404` | `RECOMMENDATION_SNAPSHOT_NOT_FOUND` — hết hạn (TTL 7 ngày), bị evict, hoặc id không do server cấp. Lỗi hạ tầng cache cũng **hạ xuống** 404 này |
| `500` | Lỗi hệ thống |

`recommendationId` **không phải UUID hợp lệ** (rỗng, sai định dạng) hoặc bị bỏ trống → **400** do
binding (`binding:"required"` trên `uuid.UUID`) — không tới được use case. Chỉ UUID hợp lệ nhưng
không có snapshot mới trả **404**. Nhánh 404 cho UUID nil trong use case chỉ áp dụng cho caller nội
bộ, không phải HTTP.

---

## 8. `POST /api/v1/ai/outfit-completions` — NEW

Hoàn thiện một tập món **đã sở hữu**. Không chạy LLM, không tạo bộ đồ mới, **không trừ quota**. Kill switch
không chặn (endpoint không phụ thuộc personalization).

**Request body** — `CompleteOutfitReq`:

```json
{
  "fashionItemIds": ["uuid", "uuid"],
  "occasion": "work",
  "styleTarget": "minimalist",
  "season": "summer",
  "weather": "hot",
  "colorTone": "light",
  "details": "..."
}
```

| Field | Kiểu | Bắt buộc | Ghi chú |
|---|---|---|---|
| `fashionItemIds` | `string[]` (uuid) | **có** | 1-10 phần tử. Toàn bộ phải thuộc **tủ đang hoạt động** của user: row `wardrobe_items` chưa xoá **và** `status = InWardrobe`. Món đã bán (`Sold`) hoặc bị khoá do hạ gói bị từ chối y như món không sở hữu. Đây là **fashionItemId**, không phải wardrobe item id |
| `occasion` | `string` | không | Tự do, như §6 |
| `styleTarget` | `string` | không | Tự do, như §6 |
| `season` | `string` | không | Enum `spring \| summer \| autumn \| winter \| all` |
| `weather` | `string` | không | Tự do |
| `colorTone` | `string` | không | Tự do |
| `details` | `string` | không | ≤ 1000 ký tự NFC |

**Response 200** — `data` = `CompleteOutfitRes`:

| Field | Kiểu | Bắt buộc | Ghi chú |
|---|---|---|---|
| `suggestions` | `array` | **luôn có** (có thể `[]`) | Cùng phần tử `OutfitSuggestionRes` như §6: `{role, reason, fashionItem}`. `[]` = tập đã đủ cấu trúc hoặc không có món catalog phù hợp |

```json
{ "suggestions": [ { "role": "footwear", "reason": "...", "fashionItem": { "id": "uuid", "category": { }, "imageUrl": "https://...", "color": "Trắng", "colorHex": "#ffffff", "style": "Tối giản" } } ] }
```

Role được xét: `fullbody` **hoặc** (`top` + `bottom`), cộng `footwear`, cộng `outerwear` khi ngữ cảnh cần
lớp ngoài (`services/commerce/internal/modules/fashion/application/usecase/personalization/completion/completion_helper.go`).

**Trạng thái**:

| Status | Khi nào |
|---|---|
| `200` | Thành công, kể cả `suggestions: []` |
| `400` | `OUTFIT_ITEM_INVALID_OR_FORBIDDEN` (một món không thuộc user — **giữ 400**, không đổi sang 403), hoặc danh sách rỗng/>10, hoặc lỗi binding |
| `401` | Thiếu/hết hạn token |
| `403` | Sai role |
| `500` | Lỗi hệ thống |

Không có `remainingQuota` trong response — client muốn cập nhật quota phải đọc lại
`GET /api/v1/subscriptions/me/daily-quota`.

---

## 9. `POST /api/v1/outfits` và `PUT /api/v1/outfits/:id` — CHANGED

`services/commerce/internal/modules/wardrobe/application/dto/outfit.go`. Request/response **giống nhau**
cho cả hai method (PUT dùng lại `SaveOutfitReq`).

**Request body**:

```json
{
  "name": "Set đồ đi làm",
  "description": "...",
  "coverImageUrl": "https://...",
  "coverPublicId": "...",
  "items": [
    {
      "fashionItemId": "uuid",
      "itemContext": "system_catalog",
      "positionX": 0,
      "positionY": 0,
      "scale": 1,
      "layerOrder": 1
    }
  ]
}
```

| Field | Kiểu | Bắt buộc | Ghi chú |
|---|---|---|---|
| `name` | `string` | **có** | ≤ 255 |
| `description` | `string` | không | |
| `coverImageUrl` | `string` | không | Phải là URL hợp lệ (`url`) |
| `coverPublicId` | `string` | không | ≤ 255 |
| `items` | `array` | **có** | |
| `items[].fashionItemId` | `string` (uuid) | **có** | |
| `items[].itemContext` | `string` | không | **MỚI**. Enum đầu vào `user_wardrobe \| brand_item \| system_catalog` (§11.4). Bỏ trống → auto-resolve theo thứ tự `user_wardrobe` → `brand_item` (eligible) → `system_catalog` |
| `items[].positionX` | `number` | không | Mặc định `0` |
| `items[].positionY` | `number` | không | Mặc định `0` |
| `items[].scale` | `number` | **có** | `>= 0.1` |
| `items[].layerOrder` | `number` (int16) | **có** | `required` → giá trị `0` bị coi là thiếu, phải `>= 1` |
| `status` | — | — | **KHÔNG tồn tại trong backend.** Xem §15.3 |

**Hành vi clone-on-save** (áp dụng cho cả POST và PUT):

1. `itemContext = "system_catalog"`: món phải có trong tủ hệ thống đang active, nếu không → 400
   `CATALOG_ITEM_UNAVAILABLE`. Món được **clone** thành `wardrobe_items` của user trong cùng transaction,
   không chạy lại pipeline phân tích ảnh.
2. Nếu user **đã sở hữu** `fashionItemId` (row `wardrobe_items` chưa xoá, **bất kỳ status**): tái sử dụng
   row đó, không clone thêm, **không** tính thêm hạn mức.
3. Outfit item sau khi lưu luôn có `item_context = user_wardrobe` (kể cả khi clone từ catalog) hoặc
   `brand_item`. Giá trị `system_catalog` **không bao giờ** được lưu vào `outfit_items`.
4. Vượt hạn mức tủ → **400** `WARDROBE_LIMIT_EXCEEDED_FOR_OUTFIT` kèm danh sách item gây vượt; **không**
   có row clone mồ côi.
5. `PUT` chỉ cập nhật `last_used_at` + phát signal cho item **mới thêm**; đổi tên không đổi item → không
   signal mới (§10).

**Response 201 (POST) / 200 (PUT)** — `data` = `OutfitRes`:

| Field | Kiểu | Bắt buộc | Ghi chú |
|---|---|---|---|
| `id` | `string` | luôn có | |
| `userId` | `string` | luôn có | |
| `name` | `string` | luôn có | |
| `description` | `string \| null` | luôn có | |
| `coverImageUrl` | `string \| null` | luôn có | |
| `status` | `number` | luôn có | `0 = Draft`, `1 = Active` (`outfitstatus`) |
| `createdAt` | `string` | luôn có | RFC3339 |
| `updatedAt` | `string` | luôn có | RFC3339 |
| `items` | `array` | có thể vắng | `omitempty` |
| `items[].id` | `string` | luôn có | |
| `items[].itemContext` | `string` | luôn có | **Chỉ** `user_wardrobe` hoặc `brand_item` |
| `items[].fashionItem` | `object` | có thể vắng | `FashionItemBriefRes` (`id, category, imageUrl, color, colorHex, style`; có thể kèm `reviewReason`, `processingErrorReason` khi món đang chờ duyệt/lỗi phân tích — client hiển thị hoặc bỏ qua, không phụ thuộc) |
| `items[].positionX` | `number` | luôn có | |
| `items[].positionY` | `number` | luôn có | |
| `items[].scale` | `number` | luôn có | |
| `items[].layerOrder` | `number` | luôn có | |
| `items[].isAvailable` | `boolean` | luôn có | **MỚI trong tài liệu client**: món đã bán/bị bỏ khỏi tủ → `false`. Client nên vẽ mờ và cảnh báo |

**Trạng thái**:

| Status | Khi nào |
|---|---|
| `201` | POST thành công |
| `200` | PUT thành công |
| `400` | `CATALOG_ITEM_UNAVAILABLE`, `WARDROBE_LIMIT_EXCEEDED_FOR_OUTFIT`, `DUPLICATE_OUTFIT_ITEM` (cùng `fashionItemId` xuất hiện 2 lần), `OUTFIT_ITEM_INVALID_OR_FORBIDDEN`, `OUTFIT_ITEM_SOLD`, hoặc lỗi binding |
| `401` | Thiếu/hết hạn token |
| `403` | `ITEM_LOCKED_DUE_TO_LIMIT` (món bị khoá do vượt hạn mức) hoặc `OUTFIT_LIMIT_REACHED` (vượt hạn mức số bộ) |
| `404` | (`PUT`/`GET`/`DELETE` `:id`) không tìm thấy outfit của user |
| `500` | Lỗi hệ thống |

---

## 10. Subscription — CHANGED

### 10.1 `GET /api/v1/subscriptions/me`

`services/account/dto/subscription/billing_dto.go` — `UserSubscriptionOverviewRes`.

| Field | Kiểu | Bắt buộc | Ghi chú |
|---|---|---|---|
| `planId` | `string` | luôn có | |
| `planName` | `string` | luôn có | |
| `planSlug` | `string` | luôn có | |
| `expiresAt` | `string \| null` | luôn có | |
| `isAutoRenewEnabled` | `boolean` | luôn có | |
| `maxWardrobeItems` | `number` | luôn có | |
| `maxOutfits` | `number` | luôn có | |
| `aiOutfitDailyQuota` | `number` | luôn có | |
| `aiChatDailyQuota` | `number` | luôn có | |
| `personalizationEnabled` | `boolean` | luôn có | **MỚI**. Dựng từ **row plan hiện tại**, không đọc `current_benefit_snapshot` |

```json
{ "planId": "uuid", "planName": "Premium Plan", "planSlug": "premium-monthly", "expiresAt": "2026-11-26T00:00:00Z", "isAutoRenewEnabled": true, "maxWardrobeItems": 300, "maxOutfits": 300, "aiOutfitDailyQuota": 30, "aiChatDailyQuota": 10, "personalizationEnabled": true }
```

Response này **không** có `status`. Nếu client cần trạng thái, xem xét lại — hiện `[MOBILE]` đang mặc định
`'ACTIVE'` (xem §15.3).

**Trạng thái**: `200`, `401`, `403` (token không phải role `user`, ví dụ token admin), `500`.

### 10.2 `GET /api/v1/subscriptions/plans` (public) và admin plan APIs

Delta duy nhất trên mọi DTO plan: thêm **`personalizationEnabled: boolean`**
(`SubscriptionPlanDTO`, `SubscriptionPlanBriefRes`).

Admin request bodies (`CreateSubscriptionPlanReq`, `UpdateSubscriptionPlanReq`):

| Field | Kiểu | Bắt buộc | Ghi chú |
|---|---|---|---|
| `personalizationEnabled` | `boolean` | không | Khi **tạo**: bỏ trống → `false`. Khi **update**: bỏ trống → giữ nguyên |

Admin cần hiển thị checkbox này trong form tạo/sửa gói; nếu không, không có cách nào bật quyền lợi cho gói
mới ngoài gọi API trực tiếp.

---

## 10b. Event RabbitMQ `wardrobe.outfit.saved` — NEW (backend-only)

**Client không có hành động nào.** Mục này tồn tại để FE/Mobile biết vì sao gu học được có thể thay đổi
sau khi user lưu bộ đồ, và để không ai đi tìm một endpoint cho nó.

| Thuộc tính | Giá trị |
|---|---|
| Topic / routing key | `wardrobe.outfit.saved` |
| Queue | `style_learning_queue` (quorum, delivery-limit 5) |
| DLQ | `style_learning_queue.dlq` |
| Phát khi | Tạo outfit thành công, hoặc `PUT /outfits/:id` **thêm item mới**. Đổi tên/ toạ độ mà tập item không đổi → không phát |
| Giao nhận | At-least-once; consumer idempotent bằng `signalId` (PK `fashion.style_signals.id`) |
| PII | Không có, ngoài internal `userId` |

Payload:

| Field | Kiểu | Bắt buộc | Ghi chú |
|---|---|---|---|
| `signalId` | `string` (uuid) | luôn có | Khóa idempotency; outbox retry dùng lại đúng ID này |
| `userId` | `string` (uuid) | luôn có | Internal user id |
| `outfitId` | `string` (uuid) | luôn có | |
| `fashionItemIds` | `string[]` (uuid) | luôn có | Với `PUT`: chỉ các item **mới thêm** |
| `source` | `string` | luôn có | Luôn là `outfit_saved` |
| `occurredAt` | `string` | luôn có | RFC3339, UTC |

```json
{
  "signalId": "uuid",
  "userId": "uuid",
  "outfitId": "uuid",
  "fashionItemIds": ["uuid", "uuid"],
  "source": "outfit_saved",
  "occurredAt": "2026-10-27T09:00:00Z"
}
```

Hệ quả phía client (không phải việc phải làm, chỉ để hiểu hành vi):

- Sau khi lưu outfit, `learned_styles`/`learned_embedding` của user cập nhật trong vòng ~60 giây và ảnh
  hưởng mềm tới thứ hạng gợi ý kế tiếp. Không có response nào cho client biết việc này đã xong.
- Signal được ghi **trong cùng transaction** với outfit (Constitution §III, outbox). Nếu publish lỗi,
  outfit cũng rollback → client nhận lỗi 5xx và **không** có outfit nửa vời.
- Feedback `saved` (§7) là tín hiệu nhẹ riêng, không thay thế event này.

---

## 11. Enum và mã trạng thái

### 11.1 `role` (outfit / suggestion)

`headwear`, `top`, `bottom`, `fullbody`, `outerwear`, `footwear`, `accessory`, `other`
(`services/commerce/internal/modules/fashion/application/mapper/recommendation.go`).

Đây là **tập giá trị chuẩn**, không phải kiểu enum đóng ở phía client: khi món đồ thiếu metadata
category, `role` do LLM khai báo được giữ nguyên (đã lowercase) thay vì map lại. Client phải có nhánh
fallback cho role lạ (hiển thị nhãn thô), đừng coi đó là response không hợp lệ.

### 11.2 `itemContext` lưu trữ (response)

`user_wardrobe`, `brand_item` (`services/commerce/internal/modules/wardrobe/domain/constants/outfititemcontext/outfit_item_context.go`).
Không bao giờ trả `system_catalog`.

### 11.3 Feedback `action`

`like`, `dislike`, `saved` — chữ thường, binding `oneof` phân biệt hoa thường.

### 11.4 `itemContext` đầu vào (request save outfit)

`user_wardrobe`, `brand_item`, `system_catalog` — binding `oneof` trên `SaveOutfitItemReq`.

### 11.5 `season`

`spring`, `summer`, `autumn`, `winter`, `all`.

### 11.6 Role slug (guard route)

`user`, `admin` (`shared/domain/constants/roleslug`).

### 11.7 Signal source

Event payload dùng chuỗi `outfit_saved`. DB `fashion.style_signals.source` dùng smallint:
`1 = outfit_saved`, `2 = feedback_like`, `3 = feedback_dislike`, `4 = feedback_saved`.
Client không thấy giá trị này.

### 11.8 Mã HTTP dùng trong phạm vi 033

`200`, `201`, `400`, `401`, `403`, `404`, `500`.

---

## 12. Tích hợp WEB (`smart-wardrobe-fe`)

### 12.1 Hiện trạng đã đối chiếu

| File `[WEB]` | Hiện trạng |
|---|---|
| `src/features/ai-stylist/types/index.ts` | `AIOutfitRecommendationRes` chỉ có `title, explanation, isFallback, remainingQuota, items`. Chưa có `recommendationId`, `suggestions`. `AIOutfitProduct` đã có `itemContext?` và `isGhost?` (từ ghost-closet) |
| `src/features/ai-stylist/api/ai.api.ts` | Chỉ có `getOutfitRecommendation` + các API chat. Chưa có feedback/completions |
| `src/app/(user)/ai-stylist/components/AIStylistClient.tsx` | Lưu state `outfitData` (mất `recommendationId` vì type không có); lưu bộ đồ bằng `useCreateOutfit()`; **không gửi `itemContext`**; không render `suggestions` |
| `src/features/outfits/types/index.ts` | `SaveOutfitItemReq` **thiếu `itemContext`**; `SaveOutfitReq` có `status?` (backend không nhận); `OutfitItemRes` có `wardrobeItem?` (backend không trả) và thiếu `isAvailable` |
| `src/features/outfits/api/outfits.api.ts` + `queries/outfits.queries.ts` | Đã đủ `createOutfit` / `updateOutfit` / `deleteOutfit` + mutation TanStack Query |
| `src/app/(user)/outfits/create/components/CreateOutfitClient.tsx` | Producer #2 — gửi `status: 0`, không gửi `itemContext` |
| `src/app/(user)/outfits/[id]/components/OutfitDetailClient.tsx` | Producer #3 — gửi `status: outfit?.status ?? 0`, không gửi `itemContext` |
| `src/app/(guest)/auth/register/preferences/components/PreferencesClient.tsx` | **Local-only**: `useState` + `router.push("/wardrobe/upload")`. Không gọi API. Nhãn hiển thị viết hoa (`Minimalist`, `Y2K`, ...), chọn thêm **bảng màu** mà backend không có |
| `src/app/(user)/wardrobe/explore/components/SystemCatalogClient.tsx` | Chỉ gọi `useInitClosetFromCatalog` (`POST /wardrobe-items/catalog-init`). Không liên quan clone-on-save |
| `src/features/subscription/types/index.ts` | `UserSubscription`/`SubscriptionPlan` **thiếu `personalizationEnabled`** |
| `src/lib/axios.ts` | Interceptor refresh + toast lỗi; baseURL `/api/v1` |
| `src/lib/api-error.ts` | `handleApiError`, `getApiErrorMessage`, `getApiErrorStatus`, `getApiErrorData` |

Canvas item `[WEB]` (`src/features/outfits/hooks/useOutfitCanvas.ts`, `CanvasItem`) **đã có** `itemContext?`
và nó được điền bởi `getBrandItemCanvasMetadata` trong `resolveCanvasOutfitItems` /
`swapCanvasItemByRole`. Nghĩa là dữ liệu đã có sẵn trên canvas, chỉ chưa được gửi lên server.

### 12.2 Các file cần sửa / thêm

| # | File `[WEB]` | Việc cần làm |
|---|---|---|
| 1 | `src/features/ai-stylist/types/index.ts` | Thêm `recommendationId: string` và `suggestions?: AIOutfitSuggestion[]` vào `AIOutfitRecommendationRes`; định nghĩa `AIOutfitSuggestion { role: FashionRole \| string; reason: string; fashionItem: {...} }`; thêm `AIOutfitFeedbackReq/Res`, `CompleteOutfitReq/Res`, `StyleProfileRes`, `UpdateStyleProfileReq`, `StyleSuggestionRes` (hoặc đặt trong feature mới, xem #2) |
| 2 | **mới** `src/features/style-profile/` (`types`, `api/style-profile.api.ts`, `queries/style-profile.queries.ts`) | `GET/PUT/DELETE /me/style-profile`, `GET /me/style-profile/suggestions`. Query key riêng, ví dụ `['style-profile']`. Mutation PUT/DELETE phải `invalidateQueries` key này và key subscription (vì `personalizationAvailable` phụ thuộc plan) |
| 3 | `src/features/ai-stylist/api/ai.api.ts` | Thêm `submitRecommendationFeedback(req)` → `POST /ai/outfit-recommendations/feedback`; `completeOutfit(req)` → `POST /ai/outfit-completions` (`api` mặc định) |
| 4 | `src/app/(user)/ai-stylist/components/AIStylistClient.tsx` | Giữ `outfitData.recommendationId`; render khối `suggestions` (ảnh + `reason` + nút "Thêm vào bộ"); gọi feedback `saved` sau khi `createOutfitMutation` thành công; gọi feedback `like`/`dislike` từ UI; gửi `itemContext` khi lưu |
| 5 | `src/features/outfits/types/index.ts` | Thêm `itemContext?: 'user_wardrobe' \| 'brand_item' \| 'system_catalog'` vào `SaveOutfitItemReq`; thêm `isAvailable?: boolean` vào `OutfitItemRes`; cân nhắc bỏ `status?` khỏi `SaveOutfitReq` (backend bỏ qua) và sửa `wardrobeItem?` → `fashionItem?` |
| 6 | `src/app/(user)/outfits/create/components/CreateOutfitClient.tsx` | Producer #2: truyền `itemContext` nếu canvas item có (món lấy từ tủ → `user_wardrobe` hoặc bỏ trống) |
| 7 | `src/app/(user)/outfits/[id]/components/OutfitDetailClient.tsx` | Producer #3: như #6. Lưu ý PUT chỉ phát signal cho món **mới thêm** |
| 8 | `src/app/(guest)/auth/register/preferences/components/PreferencesClient.tsx` | Gọi `PUT /me/style-profile` thay vì chỉ local state. Map nhãn hiển thị → value chữ thường (`Minimalist` → `minimalist`, `Y2K` → `y2k`, `Streetwear` → `streetwear`, ...). Bỏ qua lỗi 403 một cách im lặng (user free vẫn đi tiếp được) |
| 9 | `src/features/subscription/types/index.ts` | Thêm `personalizationEnabled?: boolean` vào `UserSubscription` và `SubscriptionPlan` |

### 12.3 Ghi chú triển khai cho WEB

- **`recommendationId` phải được lưu cùng `outfitData`** và truyền xuống component con. Nếu user bấm
  "thích" sau khi đã tạo bộ mới, `recommendationId` cũ trỏ tới snapshot khác — mỗi lần gọi
  `/ai/outfit-recommendations` là một snapshot mới.
- **`suggestions` là `omitempty`**: dùng `outfitData.suggestions?.length` chứ không `outfitData.suggestions.length`.
- **Feedback `saved`**: gọi sau khi `createOutfit` trả 201. Đây là tín hiệu nhẹ (0.25) nên gửi trùng với
  `wardrobe.outfit.saved` là chủ ý, không phải bug. Nếu `createOutfit` lỗi thì **không** gửi `saved`.
- **`itemContext` khi thêm món gợi ý vào canvas**: món `suggestions[]` chưa có trên canvas. Khi user thêm
  một món gợi ý, đặt `itemContext: 'system_catalog'` cho canvas item đó. Nếu bỏ trống, backend vẫn
  auto-resolve và clone — nhưng khi món đã bị gỡ khỏi tủ hệ thống, client sẽ nhận
  `OUTFIT_ITEM_INVALID_OR_FORBIDDEN` (400) thay vì `CATALOG_ITEM_UNAVAILABLE` (400) dễ hiểu hơn.
- **Lỗi clone phải hiển thị nguyên văn**: `WARDROBE_LIMIT_EXCEEDED_FOR_OUTFIT` trả kèm danh sách ID gây
  vượt, `message` đã là tiếng Việt. Dùng `handleApiError(error, ...)` — nó ưu tiên `error.response.data.message`.
- **`PreferencesClient`**: backend **không** có khái niệm bảng màu hay `colorTone` trong hồ sơ gu. Ô chọn
  bảng màu hiện tại không có nơi lưu → xem §16.1.
- **`DELETE` không có `data`**: nếu thêm nút "Xoá hồ sơ gu", đọc `res.data.message` và không truy cập
  `res.data.data`.
- **`excludedStyles`**: chưa có UI. Nếu muốn "tránh phong cách", thêm một màn riêng; backend đã hỗ trợ.

---

## 13. Tích hợp MOBILE (`smart-wardrobe-mobile`)

### 13.1 Hiện trạng đã đối chiếu

| File `[MOBILE]` | Hiện trạng |
|---|---|
| `lib/features/outfit_studio/models/outfit_models.dart` | `RecommendedOutfitRes` **thiếu `recommendationId` và `suggestions`**; `RecommendedItemGroup` khớp shape nhưng `RecommendedItemRes` **thiếu `brandItem`**; `SaveOutfitItemReq` **thiếu `itemContext`**; `OutfitItemDetailModel` **thiếu `isAvailable`**; `AIOutfitRecommendationReq.toJson()` gửi **`include_brand_items`** (sai key) |
| `lib/features/outfit_studio/data/outfit_repository.dart` | Có `getAIRecommendation`, `getMyOutfits(Paginated)`, `getOutfitDetail`, `deleteOutfit`, `saveOutfit` (**chỉ POST**), `getUploadSignatureOutfit`, `getUserWardrobeItems`. **Không có `updateOutfit` (PUT)** |
| `lib/features/outfit_studio/providers/ai_outfit_provider.dart` | `AIOutfitNotifier` giữ `RecommendedOutfitRes`; `swapAlternative` **dựng lại** `RecommendedOutfitRes` bằng tay → mọi field mới phải được copy ở đây |
| `lib/features/outfit_studio/providers/outfit_studio_provider.dart` | `saveOutfit()` luôn tạo mới qua `repo.saveOutfit`; dựng `SaveOutfitItemReq` không có `itemContext` |
| `lib/features/outfit_studio/presentation/outfit_studio_screen.dart` | Gọi `loadFromAIRecommendation(res)` và `saveOutfit(...)`. Bottom sheet thay thế chỉ tìm trong **tủ đồ** |
| `lib/features/auth/presentation/preferences_screen.dart` | **Local-only**: gọi `authStateProvider.notifier.savePreferences(styles, palette)` |
| `lib/features/auth/data/auth_repository.dart` | `saveStylePreferences` chỉ ghi `SecureStorageService` (`user_style_prefs`, `user_color_palette`). Không gọi API |
| `lib/features/auth/providers/auth_provider.dart` | `savePreferences` chỉ chuyển tiếp xuống repository |
| `lib/features/wardrobe/presentation/system_catalog_screen.dart` | Dùng `systemCatalogProvider`; luồng chỉ là **"thêm vào tủ"**, không lưu outfit |
| `lib/features/wardrobe/data/wardrobe_repository.dart` | Có `getSystemCatalogItems`, `initClosetFromCatalog` |
| `lib/features/profile/data/payment_repository.dart` | `getMySubscription`, `getDailyQuota`, `getSubscriptionPlans`. `UserSubscriptionModel`/`SubscriptionPlanModel` **thiếu `personalizationEnabled`** |
| `lib/core/network/api_client.dart` | Dio + interceptor refresh single-flight + `onGlobalForcedLogout` |

### 13.2 Hai mô hình recommendation song song

| Mô hình | File | Có được API điền? | Kết luận |
|---|---|---|---|
| Studio | `lib/features/outfit_studio/models/outfit_models.dart` (`RecommendedOutfitRes`, `RecommendedItemGroup`, `RecommendedItemRes`, `RecommendedFashionItemBrief`) | **Có** — `OutfitRepository.getAIRecommendation` map trực tiếp từ `data` của `POST /ai/outfit-recommendations` | **Chuẩn hoá trên mô hình này** |
| Stylist (chat) | `lib/features/stylist/models/stylist_models.dart` (`OutfitRecommendationModel`, `OutfitRecommendationItem`) | **Không** — `ChatMessageModel.fromJson` đọc `json['outfitRecommendation']` và `json['suggestedItems']`, nhưng `ChatMessageRes` của backend chỉ có `id, sender, content, createdAt` | **Không mở rộng**. Model này chỉ là dữ liệu chết cho tới khi chat trả outfit |

→ Mọi field mới của 033 (`recommendationId`, `suggestions`, `itemContext`) chỉ thêm vào bộ Studio. Nếu sau
này chat cần hiển thị suggestion, tái sử dụng `OutfitSuggestion` của Studio thay vì tạo bản sao thứ ba.

### 13.3 Các file cần sửa / thêm

| # | File `[MOBILE]` | Việc cần làm |
|---|---|---|
| 1 | `lib/features/outfit_studio/models/outfit_models.dart` | `RecommendedOutfitRes` thêm `recommendationId` (mặc định `''`) và `suggestions` (mặc định `const []`); class mới `OutfitSuggestion { role, reason, fashionItem }`; `SaveOutfitItemReq` thêm `itemContext` (nullable, chỉ serialize khi khác null); `RecommendedItemRes` thêm `brandItem` (nullable — nếu không thêm, món brand hiển thị rỗng); `OutfitItemDetailModel` thêm `isAvailable` (mặc định `true`); sửa `include_brand_items` → `includeBrandItems` |
| 2 | `lib/features/outfit_studio/data/outfit_repository.dart` | Thêm `updateOutfit(String id, SaveOutfitReq req)` → `PUT /outfits/$id` (dùng `_apiClient.dio`, bọc `on DioException` + `_extractErrorMessage`); thêm `submitRecommendationFeedback`, `completeOutfit` |
| 3 | **mới** `lib/features/style_profile/` (`data/style_profile_repository.dart`, `models/style_profile_models.dart`, `providers/style_profile_provider.dart`) | Repository bọc `ApiClient` (constructor `ApiClient? apiClient` như các repo khác để test inject được): `getProfile`, `updateProfile`, `deleteProfile`, `getSuggestions`. Provider Riverpod tương ứng |
| 4 | `lib/features/outfit_studio/providers/ai_outfit_provider.dart` | `swapAlternative` phải copy `recommendationId` và `suggestions` khi dựng lại `RecommendedOutfitRes` (nếu không, feedback sau khi đổi món sẽ mất id); thêm method gửi feedback |
| 5 | `lib/features/outfit_studio/providers/outfit_studio_provider.dart` | Truyền `itemContext` vào `SaveOutfitItemReq` (lấy từ canvas/nguồn món); thêm `updateOutfit` (chế độ sửa) dùng repository mới |
| 6 | `lib/features/outfit_studio/presentation/outfit_studio_screen.dart` | Render `suggestions` (bottom sheet/dải ngang) + gọi feedback `saved` sau `saveOutfit` thành công |
| 7 | `lib/features/auth/presentation/preferences_screen.dart` | Gọi style-profile API thay vì (hoặc bên cạnh) lưu local. Map nhãn → value chữ thường. Xử lý 403 |
| 8 | `lib/features/auth/data/auth_repository.dart` | Giữ `saveStylePreferences` như cache offline nếu muốn, nhưng **nguồn sự thật** phải là API. Không xoá hàm nếu màn khác còn dùng |
| 9 | `lib/features/profile/models/user_profile_models.dart` | Thêm `personalizationEnabled` (mặc định `false`) vào `UserSubscriptionModel` và `SubscriptionPlanModel` |

### 13.4 Ghi chú triển khai cho MOBILE

- **Luôn dùng `ApiClient.dio`** cho mọi method mới. `ApiClient` giữ interceptor refresh single-flight và
  gọi `onGlobalForcedLogout`; tạo `Dio()` trực tiếp sẽ mất cả hai và user sẽ bị "đăng nhập ma" khi token
  hết hạn.
- **`includeBrandItems`** hiện là bug: key sai (`include_brand_items`) → backend không nhận, và
  `AIOutfitNotifier.generateOutfit` cũng không set `includeBrandItems` (luôn `null` → bị bỏ). Sửa key là
  đủ, nhưng phải thêm UI toggle nếu muốn tính năng brand hoạt động (xem §16.4).
- **`recommendationId` phải sống qua `swapAlternative`** — đây là chỗ dễ mất nhất vì
  `RecommendedOutfitRes` được dựng lại thủ công.
- **Không có PUT nào cho outfit hiện tại**: `OutfitStudioNotifier.saveOutfit` luôn tạo mới. Trước khi thêm
  nút "sửa bộ đồ", phải thêm `updateOutfit` ở repository (§13.3 #2). UI sửa hiện chỉ có ở `[WEB]`.
- **Feedback lỗi 404 là bình thường**: snapshot hết hạn sau 7 ngày. Bắt `DioException` và hiện
  `message` từ backend thay vì chặn UI.
- **`itemContext` ở mobile hiện gần như luôn là `user_wardrobe`** vì drawer chỉ có món trong tủ. Chỉ cần
  truyền khi thêm món từ `suggestions` (catalog) — nếu làm luồng đó, set `'system_catalog'`.
- **Completions**: chưa có UI chọn nhiều món + "hoàn thiện". Đề xuất: từ `outfit_detail_page.dart` hoặc
  Studio, lấy `fashionItemId` của các món đang có → gọi `POST /ai/outfit-completions`. Không trừ quota,
  nên có thể gọi không cần hỏi.

---

## 14. Tương thích và xử lý lỗi

### 14.1 Tương thích

- Mọi field mới trong JSON là additive; client cũ bỏ qua được (`SC-011`).
- `suggestions` có `omitempty` ở recommendation → response cũ **không có key**; client mới phải chịu được
  `undefined`/`null`.
- `itemContext` trong save outfit là optional; client cũ không gửi vẫn auto-resolve như trước.
- Proto `UserSubscriptionOverview.personalization_enabled = 11` additive. Server cũ trả `false` → tính
  năng tắt, không lỗi. Deploy Account trước, Commerce sau.
- Không đổi tên/kiểu field cũ, không đổi status code hiện có.

### 14.2 Cache / default

| Chủ đề | Hành vi |
|---|---|
| `GET /me/style-profile` chưa có row | Không tạo row; trả `styles: []`, `excludedStyles: []`, `enabled: true`, `hasEmbedding: false`, `updatedAt: null` |
| `GET /me/style-profile/suggestions` chưa đủ dữ liệu | `200`, `data = {"styles": [], "sampleSize": 0}`, thông báo ở `message` |
| Redis `personalization:rec:{id}` | TTL 7 ngày; hết hạn → feedback trả 404 |
| Redis `personalization:catalog:v1` | TTL 5 phút; lỗi cache → degrade thành miss, request vẫn chạy |
| Kill switch `personalization.enabled = false` | GET profile/suggestions vẫn trả dữ liệu; PUT/DELETE 403 `PERSONALIZATION_DISABLED`; recommendation bỏ qua gu nhưng vẫn trả `suggestions`; worker ack và không ghi signal; completions vẫn chạy |
| Entitlement hết hạn | Đọc được, không sửa, không áp dụng; dữ liệu giữ nguyên |

### 14.3 Xử lý lỗi theo nền tảng

**[WEB]** `src/lib/axios.ts` + `src/lib/api-error.ts`:

- Interceptor đã bật refresh tự động cho `401` (và vài trường hợp `500` có message "Vui lòng đăng nhập").
  Request mới **không cần** xử lý refresh tay.
- Interceptor tự toast cho `403`/`429` và `>= 500` (trừ khi `silent: true`), nên mutation nào cũng gọi
  `handleApiError` thì có thể **toast hai lần** với 403. Với các lỗi 403 nghiệp vụ của 033
  (`PERSONALIZATION_NOT_AVAILABLE`, `FEEDBACK_FORBIDDEN`), dùng `getApiErrorMessage(error, fallback)` để
  lấy message và tự quyết định hiển thị (inline banner thay vì toast), tránh trùng.
- `handleApiError` **bỏ qua** lỗi mạng và `>= 500` (đã có interceptor lo) — đúng như mong đợi.
- `getApiErrorStatus(error)` để phân nhánh `404` (snapshot hết hạn) và `403` (cần nâng cấp gói).

**[MOBILE]** `lib/core/network/api_client.dart` + pattern `_extractErrorMessage` trong các repository:

- `ApiClient` refresh token một lần (single-flight), retry request gốc; thất bại → `onGlobalForcedLogout`.
  Repository mới phải dùng `_apiClient.dio` (xem §13.4).
- `_extractErrorMessage` hiện đọc `message` → `detail` → `title` từ body lỗi rồi mới tới `e.message`. Đây là
  đúng thứ tự cho envelope `apperror` của backend (không có `detail` trong envelope 033, nhưng giữ nguyên
  thứ tự không hại).
- Message lỗi đã là tiếng Việt từ backend; **không** map lại sang câu chung, chỉ fallback khi body rỗng.

---

## 15. Đã đối chiếu

### 15.1 Số liệu

| Hạng mục | Số lượng |
|---|---|
| Endpoint HTTP được liệt kê | **14** |
| Endpoint HTTP thay đổi (CHANGED) | **8** |
| Endpoint HTTP mới (NEW) | **6** |
| Endpoint bị xoá (REMOVED) | **0** |
| Event RabbitMQ được liệt kê | **1** (`wardrobe.outfit.saved`, backend-only) |
| Bảng field trong §2–§10b | **15** bảng / **104** dòng field |
| Bảng mã trạng thái theo endpoint | **5** bảng / **28** dòng |
| Bảng thuộc tính event (§10b) | **1** bảng / **6** dòng |
| Bộ enum đã kiểm | **7** (§11.1–§11.7) |
| Mã HTTP đã kiểm | **7** (§11.8) |
| File backend đã đọc | **38** |
| File client đã đọc | **32** (`[WEB]` 18, `[MOBILE]` 14) |

Phép đếm: mỗi dòng dữ liệu (không tính dòng tiêu đề và dòng phân cách) trong bảng. Bảng có tiêu đề
`| Field |` / `| Đường dẫn field |` được tính là bảng field; bảng có tiêu đề `| Status |` được đếm riêng
thành "bảng mã trạng thái"; bảng `| Thuộc tính |` của §10b đếm riêng. Tổng cộng §2–§10b có **21 bảng /
138 dòng**.

### 15.2 File backend đã đọc để viết tài liệu này

- `specs/033-personalized-styling/contracts/personalization-api.md`
- `specs/033-personalized-styling/quickstart.md`
- `specs/033-personalized-styling/data-model.md`
- `services/commerce/internal/modules/fashion/application/dto/style_profile.go`
- `services/commerce/internal/modules/fashion/application/dto/outfit_completion.go`
- `services/commerce/internal/modules/fashion/application/dto/recommendation.go`
- `services/commerce/internal/modules/fashion/application/dto/chat.go`
- `services/commerce/internal/modules/fashion/application/mapper/style_profile.go`
- `services/commerce/internal/modules/fashion/application/mapper/recommendation.go`
- `services/commerce/internal/modules/fashion/application/errors/personalization_errors.go`
- `services/commerce/internal/modules/fashion/application/errors/recommendation_errors.go`
- `services/commerce/internal/modules/fashion/application/errors/chat_errors.go`
- `services/commerce/internal/modules/fashion/application/usecase/personalization/profile/profile_uc.go`
- `services/commerce/internal/modules/fashion/application/usecase/personalization/feedback/feedback_uc.go`
- `services/commerce/internal/modules/fashion/application/usecase/personalization/completion/completion_uc.go`
- `services/commerce/internal/modules/fashion/application/usecase/personalization/completion/completion_helper.go`
- `services/commerce/internal/modules/fashion/presentation/handler/style_profile_handler.go`
- `services/commerce/internal/modules/fashion/presentation/handler/fashion_ai_handler.go`
- `services/commerce/internal/modules/fashion/presentation/router/router.go`
- `services/commerce/internal/modules/wardrobe/application/dto/outfit.go`
- `services/commerce/internal/modules/wardrobe/application/dto/outfit_event.go`
- `services/commerce/internal/modules/wardrobe/application/errors/errors.go`
- `services/commerce/internal/modules/wardrobe/application/usecase/outfit/outfit_uc_resolve.go`
- `services/commerce/internal/modules/wardrobe/domain/constants/outfititemcontext/outfit_item_context.go`
- `services/commerce/internal/modules/wardrobe/domain/constants/outfitstatus/outfitstatus.go`
- `services/commerce/internal/modules/wardrobe/infrastructure/constants/recommendation.go`
- `services/commerce/internal/modules/wardrobe/presentation/handler/outfit_handler.go`
- `services/commerce/internal/modules/wardrobe/presentation/router/router.go`
- `services/account/dto/subscription/billing_dto.go`
- `services/account/dto/subscription/contract_dto.go`
- `services/account/internal/modules/subscription/presentation/router/router.go`
- `shared/presentation/response.go`
- `shared/presentation/middleware/error_handler.go`
- `shared/application/constants/apperror/error.go`
- `shared/pkg/utils/validation/validation.go`
- `shared/domain/constants/roleslug/roleslug.go`
- `shared/application/constants/eventconstants/eventconstants.go`
- `shared/config/config.go` (khối `personalization`)

### 15.3 Sai lệch tìm thấy giữa client và spec/backend

| # | Nơi | Sai lệch | Mức |
|---|---|---|---|
| 1 | `[MOBILE]` `outfit_models.dart` | `AIOutfitRecommendationReq.toJson()` serialize **`include_brand_items`** trong khi backend bind `includeBrandItems` → cờ brand không bao giờ tới server | Thấp (tính năng brand trên mobile đang tắt âm thầm) |
| 2 | `[WEB]` `features/outfits/types/index.ts` | `SaveOutfitReq.status` được gửi (`status: 0/1`) nhưng `wardrobe/dto/outfit.go` **không có** field `status` → bị bỏ qua, outfit LUÔN lưu theo mặc định của backend | Thấp (gây hiểu nhầm "AI curated" vs "tự tạo") |
| 3 | `[WEB]` `features/outfits/types/index.ts` | `OutfitItemRes.wardrobeItem?` không tồn tại trong response; backend trả `fashionItem`, `itemContext`, `isAvailable` (thiếu trong type) | Thấp |
| 4 | `[WEB]` `PreferencesClient.tsx` + `[MOBILE]` `preferences_screen.dart` | Cả hai thu thập **bảng màu** (`Neutral/Earth/Monochrome/Vibrant`) nhưng hồ sơ gu 033 chỉ có `styles`, `excludedStyles`, `enabled` — không có nơi lưu | **Cần quyết định** (§16.1) |
| 5 | `[MOBILE]` `UserSubscriptionModel` | Backend `/subscriptions/me` không trả `status`, nhưng model mặc định `'ACTIVE'` và `isPremium` dựa vào nó | Thấp (pre-existing) |
| 6 | `[WEB]` `subscription/types` | Type có cả camelCase và PascalCase (`planSlug?` và `PlanSlug?`) nhưng backend chỉ trả camelCase | Thấp (pre-existing) |
| 7 | `[MOBILE]` `stylist_models.dart` | `ChatMessageModel.fromJson` đọc `outfitRecommendation`/`suggestedItems` mà `ChatMessageRes` không có → luôn null | Thấp (dead code) |

### 15.4 Điều không xác minh được bằng mã nguồn ("chưa xác định")

| # | Chi tiết | Trạng thái | File cần kiểm để chốt |
|---|---|---|---|
| 1 | Giá trị thực tế của `personalization.max_styles` trên từng môi trường | **chưa xác định** — code chỉ cho biết default 4 | file cấu hình triển khai (YAML/ENV của từng môi trường) đọc bởi `shared/config/config.go`, mục `personalization` |
| 2 | Số suggestion tối đa thực tế mỗi response | **chưa xác định** — contract ghi 1/role, 3/response; code đọc từ config | `shared/config/config.go` (`suggestion_max_per_role`, `suggestion_max_total`) và file cấu hình môi trường |
| 3 | Nội dung chính xác của prompt LLM khi áp gu | **chưa xác định** — chỉ ảnh hưởng `explanation`, không ảnh hưởng hợp đồng | `services/commerce/internal/modules/fashion/application/usecase/ai/recommendation/synthesis/` |
| 4 | Khối lượng dữ liệu thật của `suggestions` (bao nhiêu món catalog đang active) | **chưa xác định** — phụ thuộc dữ liệu seed/production | bảng `wardrobe.wardrobe_items` nơi `item_type = 1` (system catalog) |
| 5 | UI/UX mong muốn cho `excludedStyles` | **chưa xác định** — cả hai client chưa có nhánh UI nào | cần quyết định sản phẩm, xem §16.2 |
| 6 | Có cần hiển thị `personalizationEnabled` trên màn Pricing hay chỉ dùng nội bộ | **chưa xác định** — chưa client nào đọc field này | `[WEB]` `src/features/subscription/components/PricingCard.tsx`, `[MOBILE]` màn danh sách gói |

### 15.5 Đối chiếu lại tại close-out (T091)

Ngày 2026-10-27, sau khi mã nguồn backend ổn định. Phạm vi đếm: mã nguồn backend + hai repo client;
**không** chạy runtime, **không** đo p95 (xem `docs/development/technical-notes.md`).

| Hạng mục | Số lượng | Kết quả |
|---|---|---|
| Tag `json:"…"` trong 5 file DTO của 033 (`style_profile.go`, `outfit_completion.go`, `recommendation.go`, `outfit.go`, `outfit_event.go`) | **110** | **96** tag client thấy đều có trong guide; **14** tag của `ParsedIntent` là **nội bộ**, ghi rõ ở dưới |
| Field `personalizationEnabled` (Account) | **1 field / 5 struct** | Có trong guide; các field cũ của DTO plan là delta-exempt, không liệt kê lại |
| Nhóm enum (§11.1–§11.7) | **7 nhóm / 28 mục** (26 giá trị phân biệt; `user_wardrobe`/`brand_item` dùng chung input + output) | **0 lệch** so với constant Go; role ghi rõ là tập mở |
| Route HTTP (§1.1) | **14** (6 NEW + 8 CHANGED + 0 REMOVED) | **14/14 khớp** router Gin; `:param` ⇄ `{param}` đã chuyển trước khi so |
| Event RabbitMQ | **1** (`wardrobe.outfit.saved`) | Khớp topic/queue/payload |
| Mã HTTP guide có nêu (§11.8) | **7** (`200, 201, 400, 401, 403, 404, 500`) | **7/7** handler có thể trả; đã kiểm từng constructor lỗi |
| Sai lệch đã sửa trong lần này | **6** | Liệt kê dưới |

**Chuỗi nội bộ không lộ ra client:** `feedback_like`, `feedback_dislike`, `feedback_saved`,
`outfit_saved` là khóa ánh xạ trong `learning_helper.go`; client chỉ thấy `action` (`like` /
`dislike` / `saved`) và — với event — `source = "outfit_saved"`.

**Nội bộ, không nằm trong hợp đồng client:** `dto.ParsedIntent` (14 tag: `semanticQuery`,
`lexicalTerms`, `occasion`, `styleTarget`, `colorTone`, `positiveConstraints`, `negativeConstraints`,
`excludedStyles`, `excludedColorTones`, `excludedWeather`, `personalStyle`, `learnedStyle`,
`avoidStyle`, `learnedStyleWeight`) là cấu trúc intent dùng trong pipeline, không có endpoint nào trả
nó. Ngoài ra `OutfitBriefRes` (list response, không đổi) và phần field cũ của `SubscriptionPlanDTO` /
`SubscriptionPlanBriefRes` / `CreateSubscriptionPlanReq` / `UpdateSubscriptionPlanReq` không phải bề
mặt 033 nên chỉ ghi delta `personalizationEnabled`.

**Sai lệch đã sửa:**

1. §1.3 — `title` của envelope lỗi không phải `"Forbidden"` mà là tiếng Việt
   (`NewForbidden` → `"Không có quyền truy cập"`); bổ sung `debugMessage`/`stackTrace` chỉ có ở
   non-production.
2. §7 — `recommendationId` rỗng/sai định dạng bị **binding trả 400**, không phải 404; 404 chỉ khi
   UUID hợp lệ nhưng không có snapshot.
3. §8 — `fashionItemIds` của completions chỉ nhận món **đang hoạt động** (`InWardrobe`, chưa xoá),
   không phải "mọi status" như bản trước.
4. §10.1 — bổ sung `403` (token không phải role `user`) vào danh sách trạng thái của
   `GET /subscriptions/me`.
5. §11.1 — ghi rõ `role` là **tập giá trị chuẩn**, không phải enum đóng: item thiếu category có thể
   mang role do LLM khai báo.
6. §9 — `FashionItemBriefRes` có thể kèm `reviewReason` / `processingErrorReason`; §13.1/§13.3 ghi
   thêm hai khoảng trống phía mobile: `RecommendedItemRes` thiếu `brandItem`,
   `OutfitItemDetailModel` thiếu `isAvailable`.

**Chưa kiểm được:** hành vi runtime của hai client (không chạy web/mobile); giá trị thực tế của
`personalization.max_styles` / `suggestion_max_total` / `suggestion_max_per_role` trên từng môi
trường (chỉ xác nhận default trong code + `configs/config.yaml`); p95 và tỷ lệ `isFallback` của
endpoint recommendation.

---

## 16. Điểm cần quyết định

### 16.1 Bảng màu trong màn Preferences

Backend không có field lưu bảng màu/`colorTone` trong hồ sơ gu. Ba lựa chọn:

1. Bỏ khối chọn bảng màu khỏi màn onboarding (đơn giản nhất, khớp backend).
2. Giữ UI nhưng chỉ lưu local (như hiện tại), ghi rõ là "chưa áp dụng vào gợi ý".
3. Mở rộng hồ sơ gu (thêm field mới) — **ngoài phạm vi 033**, cần feature riêng.

### 16.2 `excludedStyles` chưa có UI

Backend đã hỗ trợ "phong cách muốn tránh". Cần quyết định: thêm màn cài đặt gu (web: settings; mobile:
profile) hay để trống trong v1.

### 16.3 Nơi đặt màn "Hồ sơ gu" sau onboarding

`PreferencesClient` là bước 2/2 của luồng đăng ký. Sau đó user không còn chỗ nào sửa gu (web) và chỉ có
`preferences_screen` một lần (mobile). Cần quyết định có thêm entry point trong Settings/Profile hay không.

### 16.4 Brand toggle trên mobile

`[MOBILE]` không có UI bật `includeBrandItems` (khác `[WEB]` đã có switch ghost-closet). Sửa key JSON là
chưa đủ để tính năng hoạt động — cần quyết định có làm UI không.

### 16.5 Chuẩn hoá model recommendation trên mobile

Đề xuất (đã nêu ở §13.2): chuẩn hoá trên `outfit_studio/models/outfit_models.dart`, để nguyên
`stylist/models/stylist_models.dart`. Nếu team muốn hợp nhất, đó là refactor riêng, không thuộc T084.

### 16.6 `status` trong `SaveOutfitReq` của web

Backend không nhận `status`. Hai lựa chọn: bỏ khỏi type (khuyến nghị), hoặc yêu cầu backend thêm field để
phân biệt outfit do AI tạo và do user tự tạo (thay đổi hợp đồng, cần feature mới).
