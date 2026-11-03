# Contract Delta - Cá nhân hoá phối đồ và gợi ý bổ sung

**Feature**: `033-personalized-styling` | **Date**: 2026-10-27

Tất cả thay đổi là additive. Response envelope giữ nguyên `shared_pres.APIResponse`; tài liệu này chỉ mô
tả phần `data`.

## 1. HTTP - Hồ sơ gu (Commerce, role `user`)

### 1.1 `GET /api/v1/me/style-profile`

Trả hồ sơ hiện tại. Nếu chưa có row, trả giá trị mặc định (không tạo row).

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

- `personalizationAvailable`: lấy từ `UserSubscriptionContractDTO.personalizationEnabled`.
- `hasEmbedding`: `style_embedding IS NOT NULL`.
- Người dùng hết quyền lợi vẫn xem được; `personalizationAvailable = false`.

### 1.2 `PUT /api/v1/me/style-profile`

Partial update (field bỏ trống = giữ nguyên; mảng rỗng = xoá danh sách).

```json
{
  "styles": ["minimalist", "elegant"],
  "excludedStyles": ["streetwear"],
  "enabled": true
}
```

- `styles`, `excludedStyles`: mảng chuỗi tự do; trim, dedupe không phân biệt hoa thường, mỗi giá trị ≤ 50
  ký tự, tối đa `maxStyles` (mặc định 4).
- `enabled`: bật/tắt áp dụng.
- Response giống `GET`.
- Lỗi: `403 PERSONALIZATION_NOT_AVAILABLE`; `403 PERSONALIZATION_DISABLED`; `400 STYLE_LIMIT_EXCEEDED`;
  `400 STYLE_VALUE_INVALID`.
- Embedding sinh fail-soft: response vẫn 200 với `hasEmbedding = false` nếu provider lỗi.

### 1.3 `DELETE /api/v1/me/style-profile`

Xoá row hồ sơ (idempotent). `200 OK` với `APIResponse{message, data: null}` (theo convention
`shared_pres.Success` của repo). Đây là thao tác ghi: người dùng không có quyền lợi nhận
`403 PERSONALIZATION_NOT_AVAILABLE`; người dùng có quyền nhưng đã tắt cá nhân hoá vẫn được xoá (xoá dữ
liệu là quyền của chủ sở hữu).

### 1.4 `GET /api/v1/me/style-profile/suggestions`

Gợi ý gu từ tủ đồ hiện có (cold start). Không tự áp dụng.

```json
{
  "styles": [
    { "value": "minimalist", "itemCount": 12, "ratio": 0.4 },
    { "value": "elegant", "itemCount": 7, "ratio": 0.23 }
  ],
  "sampleSize": 30
}
```

- `ratio` = `itemCount / sampleSize` (làm tròn 2 chữ số).
- `sampleSize` = số active item có style không rỗng.
- Không đủ dữ liệu: `data` là `{"styles": [], "sampleSize": 0}`; thông báo "Chưa đủ dữ liệu..." nằm ở
  `message` của envelope hiện có, không thêm field `message` vào `data`.
- `403 PERSONALIZATION_NOT_AVAILABLE` nếu không có quyền lợi.

## 2. HTTP - Gợi ý phối đồ (Commerce, changed)

### 2.1 `POST /api/v1/ai/outfit-recommendations`

Request không đổi. Response thêm hai field:

```json
{
  "recommendationId": "5b0c9f54-7d61-4f7a-9a4e-9f1f7f4f3a21",
  "title": "Bộ đồ công sở tối giản",
  "explanation": "...",
  "items": [ ... ],
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

- `recommendationId`: luôn có; dùng cho feedback. Trường này có mặt từ US1; endpoint feedback thuộc US6.
- `suggestions`: `omitempty`; tối đa 1 món/role, 3 món/response; không tính vào `items`.
- Không thay đổi `title`, `explanation`, `items`, `isFallback`, `remainingQuota`.

### 2.2 `POST /api/v1/ai/outfit-recommendations/feedback`

```json
{ "recommendationId": "5b0c9f54-7d61-4f7a-9a4e-9f1f7f4f3a21", "action": "like" }
```

- `action`: `like | dislike | saved`.
- Response: `200 {"accepted": true}`.
- Idempotent theo `(user, recommendationId, action)`.
- Lỗi: `404 RECOMMENDATION_SNAPSHOT_NOT_FOUND` (hết hạn/không tồn tại);
  `403 FEEDBACK_FORBIDDEN` (snapshot thuộc user khác).
- `saved` là tín hiệu nhẹ (0.25) để tránh cộng hai lần với sự kiện lưu outfit.

### 2.3 `POST /api/v1/ai/outfit-completions`

Hoàn thiện một tập món đã sở hữu. Không gọi LLM, không trừ quota.

Request:

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

Response:

```json
{
  "suggestions": [
    { "role": "footwear", "reason": "...", "fashionItem": { ... } }
  ]
}
```

- `fashionItemIds`: 1-10 món; toàn bộ phải thuộc người dùng.
- Chỉ trả gợi ý cho role còn thiếu/chưa phù hợp; nếu đủ cấu trúc → `suggestions: []`.
- Lỗi: `400 OUTFIT_ITEM_INVALID_OR_FORBIDDEN` nếu có món không thuộc user (giữ nguyên hành vi hiện tại
  của `ErrOutfitItemInvalidOrForbidden`; không đổi sang 403).

## 3. HTTP - Lưu/Cập nhật outfit (Wardrobe, changed)

`POST /api/v1/outfits` và `PUT /api/v1/outfits/:id`:

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

- `itemContext` optional: `user_wardrobe | brand_item | system_catalog`. Bỏ trống → auto-resolve theo thứ
  tự wardrobe → brand eligible → system catalog.
- Món `system_catalog` được clone thành `wardrobe_items` của user; outfit item lưu cùng `fashion_item_id`
  với `item_context = user_wardrobe`.
- Nếu người dùng đã sở hữu `fashionItemId` (row `wardrobe_items` chưa xoá, bất kỳ status), hệ thống tái
  sử dụng row đó; không clone thêm và không tính thêm hạn mức.
- Lỗi:
  - `400 CATALOG_ITEM_UNAVAILABLE` (bị gỡ/không active/không có fashion item).
  - `400 WARDROBE_LIMIT_EXCEEDED_FOR_OUTFIT` kèm danh sách item gây vượt.
  - `400 DUPLICATE_OUTFIT_ITEM` nếu cùng `fashionItemId` xuất hiện hai lần.
  - `400 OUTFIT_ITEM_INVALID_OR_FORBIDDEN` (món không thuộc tủ/brand không hợp lệ).
  - `400 OUTFIT_ITEM_SOLD` (món đã bán).
  - `403 ITEM_LOCKED_DUE_TO_LIMIT` (món bị khoá do vượt hạn mức).
- Không có thay đổi phá vỡ response hiện tại.

## 4. HTTP - Subscription (Account, changed)

### 4.1 `GET /api/v1/subscriptions/me`

Thêm `personalizationEnabled` (bool):

```json
{
  "planId": "uuid",
  "planName": "Premium Plan",
  "planSlug": "premium-monthly",
  "expiresAt": "2026-11-26T00:00:00Z",
  "isAutoRenewEnabled": true,
  "maxWardrobeItems": 300,
  "maxOutfits": 300,
  "aiOutfitDailyQuota": 30,
  "aiChatDailyQuota": 10,
  "personalizationEnabled": true
}
```

### 4.2 `GET /api/v1/subscriptions/plans` và admin plan APIs

- Public plan brief và admin plan DTO thêm `personalizationEnabled`.
- `POST /api/v1/admin/subscriptions/plans` và `PUT /api/v1/admin/subscriptions/plans/:id` nhận
  `personalizationEnabled` (optional boolean; mặc định false khi tạo).

## 5. gRPC - `contracts/subscription/v1/subscription.proto`

```proto
message UserSubscriptionOverview {
  string plan_id = 1;
  string plan_name = 2;
  string plan_slug = 3;
  google.protobuf.Timestamp expires_at = 4;
  bool has_expires_at = 5;
  bool is_auto_renew_enabled = 6;
  int32 max_wardrobe_items = 7;
  int32 max_outfits = 8;
  int32 ai_outfit_daily_quota = 9;
  int32 ai_chat_daily_quota = 10;
  bool personalization_enabled = 11; // new, additive
}
```

- Account `to_proto.go` map field; Commerce `transport/mapper.go` đọc field.
- Server cũ trả field mặc định `false`; client mới đọc `false` → tính năng tắt, không lỗi.
- `make buf-gen` và `make buf-lint` phải xanh.

## 6. RabbitMQ - event `wardrobe.outfit.saved`

- Topic/routing key: `wardrobe.outfit.saved`
- Queue: `style_learning_queue` (quorum, delivery-limit 5, DLQ
  `style_learning_queue.dlq`)
- Payload:

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

- Giao ít nhất một lần. Consumer idempotent bằng `signalId`; hàng DB dùng PK `id` = `signalId`
  (`fashion.style_signals.id`).
- `UpdateOutfit` chỉ phát các item mới thêm; không phát nếu tập item không đổi.
- Không chứa PII ngoài internal user id.

## 7. Bảng mã lỗi mới

| Mã | HTTP | Khi nào |
|---|---|---|
| `PERSONALIZATION_NOT_AVAILABLE` | 403 | Gói không có quyền cá nhân hoá |
| `PERSONALIZATION_DISABLED` | 403 | Kill switch `personalization.enabled = false` chặn ghi |
| `STYLE_LIMIT_EXCEEDED` | 400 | Vượt số style tối đa |
| `STYLE_VALUE_INVALID` | 400 | Giá trị > 50 ký tự (rỗng sau trim bị bỏ qua) |
| `CATALOG_ITEM_UNAVAILABLE` | 400 | Món hệ thống bị gỡ/không hợp lệ khi lưu |
| `WARDROBE_LIMIT_EXCEEDED_FOR_OUTFIT` | 400 | Clone làm vượt hạn mức tủ |
| `DUPLICATE_OUTFIT_ITEM` | 400 | Cùng fashion item xuất hiện hai lần |
| `RECOMMENDATION_SNAPSHOT_NOT_FOUND` | 404 | Feedback hết hạn/không tồn tại |
| `FEEDBACK_FORBIDDEN` | 403 | Snapshot thuộc user khác |

Thông điệp hiển thị bằng tiếng Việt theo quy ước `apperror`. Cột "Mã" là tên constructor nội bộ, không
phải field máy đọc trong response (`apperror.Error` không có `code`); client phân nhánh theo HTTP status
+ message.

## 8. Kill switch vận hành

`personalization.enabled = false`:

- `GET /me/style-profile` và `/suggestions` vẫn trả dữ liệu (read-only) để vận hành có thể kiểm tra.
- `PUT`/`DELETE` trả `403 PERSONALIZATION_DISABLED`.
- Recommendation bỏ qua gu/learned (như profile rỗng); vẫn trả outfit + suggestions.
- Worker learning ack message và không ghi signal/profile.
- Catalog suggestions/completions vẫn hoạt động (không phụ thuộc personalization).

## 9. Tương thích

- Field mới trong JSON là additive; client cũ bỏ qua.
- `suggestions` có `omitempty`; response cũ không có key này vẫn đọc được.
- `itemContext` trong save outfit optional; client cũ không gửi vẫn auto-resolve.
- Proto field 11 additive; cache entry cũ thiếu field → `false`.
- Không đổi tên/kiểu field cũ; không đổi status code hiện có.
