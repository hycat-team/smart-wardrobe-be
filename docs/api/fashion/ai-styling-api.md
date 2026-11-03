# AI Styling API Specs

Tài liệu này mô tả API AI styling đang được triển khai (implemented) và liên quan trực tiếp đến đợt B2B2C rebuild, đặc biệt là khả năng đưa brand items hợp lệ vào gợi ý phối đồ.

---

## Flow 1: Gợi ý outfit với wardrobe items và brand items

### 1. Tạo gợi ý phối đồ
*   **Endpoint:** `POST /api/v1/ai/outfit-recommendations`
*   **Tác nhân (Actor):** Khách hàng (User).
*   **Đối tượng ảnh hưởng:** Đọc ứng viên wardrobe/fashion candidates, có thể đọc eligible brand items; ghi nhận lượt dùng/chi phí (usage/cost) theo gói đăng ký (subscription) nếu backend cấu hình.
*   **Mô tả:** Tạo gợi ý phối đồ (outfit) từ tủ đồ (wardrobe) của user. Nếu request bật `includeBrandItems`, backend có thể lấy thêm brand items hợp lệ qua Brand contract và đưa vào candidate pool.
*   **Request Body tham khảo:**
    ```json
    {
      "occasion": "work",
      "styleTarget": "minimalist",
      "season": "summer",
      "weather": "hot",
      "colorTone": "neutral",
      "details": "Cần outfit lịch sự cho buổi gặp khách hàng",
      "includeBrandItems": true
    }
    ```
*   **Response `200 OK`:**
    ```json
    {
      "recommendationId": "5b0c9f54-7d61-4f7a-9a4e-9f1f7f4f3a21",
      "title": "Set đồ Streetwear năng động",
      "explanation": "Sự kết hợp giữa hoodie tím nổi bật và quần jogger xám tạo nên phong cách đường phố thoải mái, trẻ trung.",
      "items": [
        {
          "role": "top",
          "primary": {
            "id": "fdb10007-0000-0000-0000-000000000007",
            "itemContext": "brand_item",
            "brandItem": {
              "id": "fdb20001-0000-0000-0000-000000000001",
              "brandId": "feed0001-0000-0000-0000-000000000001",
              "brandName": "Feedback Test Brand",
              "itemType": "sample",
              "name": "Digital Sample Hoodie",
              "price": null
            },
            "fashionItem": {
              "id": "fdb10007-0000-0000-0000-000000000007",
              "category": { "id": "…", "name": "Áo", "slug": "ao" },
              "imageUrl": "https://…",
              "color": "Purple",
              "colorHex": "#7c3aed",
              "style": "Streetwear",
              "material": "Fleece",
              "pattern": "Solid",
              "fit": "Oversized"
            }
          },
          "alternatives": [
            {
              "id": "…",
              "itemContext": "user_wardrobe",
              "fashionItem": {
                "id": "…",
                "category": { "id": "…", "name": "Áo", "slug": "ao" },
                "imageUrl": "https://…",
                "color": "Đen tuyền",
                "colorHex": "#000000",
                "style": "Tối giản"
              }
            }
          ]
        },
        {
          "role": "outerwear",
          "primary": {
            "id": "fdb10008-…",
            "itemContext": "brand_item",
            "brandItem": { … },
            "fashionItem": { … }
          },
          "alternatives": []
        },
        {
          "role": "bottom",
          "primary": {
            "id": "…",
            "itemContext": "user_wardrobe",
            "fashionItem": { … }
          },
          "alternatives": [ … ]
        },
        {
          "role": "footwear",
          "primary": { … },
          "alternatives": [ … ]
        }
      ],
      "isFallback": false,
      "remainingQuota": 12,
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
    **Ghi chú response:**
    - `recommendationId` (mới, 033): luôn có; dùng cho endpoint feedback. **Trước → sau:** response cũ bắt đầu bằng `title`; response nay có thêm `recommendationId` ở đầu, các field cũ giữ nguyên.
    - `suggestions` (mới, 033, `omitempty`): mảng gợi ý món trong **tủ đồ hệ thống** để lấp vai trò còn thiếu/chưa phù hợp. Tối đa 1 món/vai trò và 3 món/response; **không** nằm trong `items`, **không** tính vào outfit và **không** trừ quota. Xem mục Flow 1B bên dưới.
    - Item `itemContext = "brand_item"` là brand sample, kèm `brandItem` để FE hiển thị nút feedback.
    - Item `itemContext = "user_wardrobe"` là đồ trong tủ của user.
    - Các field zero value (`userId`, `status`, `isLocked`, `createdAt`, `isDeleted`, `lastUsedAt`, HSL/thời vụ/mô tả của fashionItem) được lược bỏ để giảm kích thước response ~70%.

### 2. Gửi feedback cho brand sample trong outfit gợi ý
*   **Endpoint:** `POST /api/v1/brand-items/:itemId/feedbacks`
*   **Tác nhân (Actor):** Khách hàng (User).
*   **Thời điểm gọi:** Sau khi frontend hiển thị kết quả `POST /api/v1/ai/outfit-recommendations`, nếu outfit có item thuộc Brand Item và `itemType = sample`, frontend có thể cho user vote/feedback ngay tại màn hình kết quả phối đồ.
*   **Đối tượng ảnh hưởng:** Tạo bản ghi `digital_sample_responses`; có thể liên kết với outfit đã lưu nếu client gửi `outfitId`.
*   **Mô tả:** API này dùng chung với Digital Sample Lab. `itemId` là id của `brand_items` sample trong kết quả gợi ý. Nếu user đã lưu gợi ý thành outfit bằng `POST /api/v1/outfits`, client nên gửi kèm `outfitId` để brand xem feedback trong bối cảnh bộ phối đồ. Nếu user feedback ngay trên kết quả AI và chưa lưu outfit, `outfitId` có thể bỏ trống.
*   **Request Body tham khảo:**
    ```json
    {
      "outfitId": "2a9de8d4-0d1c-459e-bf22-9479a9320111",
      "voteType": "would_buy",
      "rating": 5,
      "feedbackText": "Mẫu áo khoác này hợp với outfit AI vừa gợi ý, mình sẽ mua nếu sản xuất."
    }
    ```
*   **Response:**
    *   `201 Created`: Trả về phản hồi mẫu thử vừa lưu `DigitalSampleResponseRes`.
*   **Tham chiếu:** Chi tiết validation, vote type và response xem [sample-lab-api.md](../sample-lab-api.md).

### Quy tắc B2B2C
*   `includeBrandItems = true` không có nghĩa bắt buộc phải dùng brand item; AI chỉ nên dùng nếu phù hợp với outfit.
*   Fashion module gọi Brand contract để lấy danh sách brand items được phép, không tự đọc chi tiết loyalty/benefit.
*   Brand item candidate phải thỏa mãn điều kiện brand active, item active, và user có quyền truy cập hợp lệ (access) nếu item yêu cầu đặc quyền (benefit). Các benefit/feature code tham chiếu [constants.md](../brand/constants.md).
*   AI output không được tự ý inject brand item không nằm trong candidate list mà backend đã cung cấp.
*   Item trong outfit response cần giữ `itemContext = brand_item`; giá trị tham chiếu [constants.md](constants.md).
*   Nếu brand item trong response là `sample`, frontend nên cho user gửi feedback qua `POST /api/v1/brand-items/:itemId/feedbacks` ngay tại màn hình kết quả gợi ý. Nên đính kèm `outfitId` khi user đã lưu outfit đó.

---

## Flow 1B: Gợi ý bổ sung, phản hồi và hồ sơ gu (033)

Nhóm API này là **additive**: client cũ không gọi vẫn chạy bình thường. Mọi thao tác ghi hồ sơ gu yêu cầu
quyền lợi gói (`subscription_plans.personalization_enabled`) và không bị ảnh hưởng bởi hard filter.

### 1. Gửi phản hồi cho một gợi ý

*   **Endpoint:** `POST /api/v1/ai/outfit-recommendations/feedback`
*   **Actor:** Khách hàng (User).
*   **Mô tả:** Phản hồi bộ đồ vừa được gợi ý. Dùng chung usecase học gu với sự kiện lưu outfit.
*   **Request Body:**
    ```json
    { "recommendationId": "5b0c9f54-7d61-4f7a-9a4e-9f1f7f4f3a21", "action": "like" }
    ```
    -   `recommendationId` (required): giá trị nhận từ response gợi ý.
    -   `action` (required): `like | dislike | saved`. `saved` là tín hiệu nhẹ (trọng số 0.25) để không cộng
        hai lần với sự kiện lưu outfit.
*   **Response `200 OK`:** `{ "accepted": true }`.
*   **Ghi chú:**
    -   Idempotent theo `(user, recommendationId, action)`.
    -   Snapshot recommendation nằm ở Redis `personalization:rec:{recommendationId}` (TTL 7 ngày); server tự
        đọc item IDs từ snapshot, **không tin** danh sách item do client gửi lên.
    -   `404` khi snapshot hết hạn/không tồn tại; `403` khi snapshot thuộc user khác.
*   **Trước → sau:** trước 033 chưa có endpoint này (gọi trả `404` route).

### 2. Hoàn thiện một bộ đồ đã sở hữu

*   **Endpoint:** `POST /api/v1/ai/outfit-completions`
*   **Actor:** Khách hàng (User).
*   **Mô tả:** Cho biết tập món người dùng đã chọn còn thiếu vai trò nào và gợi ý món trong tủ hệ thống để
    lấp. **Không** gọi LLM, **không** tạo outfit, **không** trừ quota.
*   **Request Body:**
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
    -   `fashionItemIds` (required): 1–10 món, toàn bộ phải thuộc người dùng hiện tại.
    -   Các field còn lại optional, cùng ngữ nghĩa với repo gợi ý phối đồ.
*   **Response `200 OK`:**
    ```json
    {
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
    -   `suggestions` luôn có (mảng rỗng nếu bộ đã đủ cấu trúc hoặc không có món phù hợp).
    -   Cùng cấu trúc `OutfitSuggestionRes` như `suggestions` của endpoint gợi ý.
*   **Lỗi:** `400` nếu có món không thuộc user hoặc vượt quá 10 món.
*   **Trước → sau:** endpoint mới (033).

### 3. Hồ sơ gu phong cách

#### 3.1 `GET /api/v1/me/style-profile`

*   **Actor:** Khách hàng (User).
*   **Mô tả:** Trả hồ sơ gu hiện tại. Nếu chưa có row, trả giá trị mặc định và **không** tạo row.
*   **Response `200 OK`:**
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
    -   `personalizationAvailable`: đọc từ quyền lợi gói (`personalizationEnabled`).
    -   `hasEmbedding`: `style_embedding IS NOT NULL`; `false` khi provider lỗi nhưng hồ sơ vẫn dùng được ở
        mức từ khoá.
    -   `updatedAt`: `null` khi chưa từng lưu hồ sơ.
    -   Người dùng hết quyền lợi vẫn **xem** được hồ sơ với `personalizationAvailable = false`.

#### 3.2 `PUT /api/v1/me/style-profile`

*   **Actor:** Khách hàng (User).
*   **Mô tả:** Cập nhật một phần. Field bỏ trống = giữ nguyên; mảng rỗng = xoá danh sách.
*   **Request Body:**
    ```json
    { "styles": ["minimalist", "elegant"], "excludedStyles": ["streetwear"], "enabled": true }
    ```
    -   `styles`/`excludedStyles`: mảng chuỗi tự do; trim, dedupe không phân biệt hoa thường, mỗi giá trị
        ≤ 50 ký tự, tối đa `maxStyles` (mặc định 4).
    -   `enabled`: bật/tắt **áp dụng** gu, không xoá dữ liệu.
*   **Response `200 OK`:** giống `GET`.
*   **Lỗi:** `403` khi gói không có quyền hoặc kill switch đang tắt; `400` khi vượt số style/giá trị quá dài.
*   **Ghi chú:** embedding sinh **tại lúc PUT** (timeout cấu hình), không sinh lúc gợi ý. Embed lỗi ⇒ vẫn
    200 với `hasEmbedding = false`.

#### 3.3 `DELETE /api/v1/me/style-profile`

*   **Actor:** Khách hàng (User).
*   **Mô tả:** Xoá row hồ sơ (idempotent). `200 OK` với `data: null`.
*   **Ghi chú:** người dùng có quyền nhưng đã **tắt áp dụng gu** (`enabled = false`) vẫn được xoá; người
    dùng không có quyền nhận `403`, và kill switch của hệ thống cũng trả `403`.

#### 3.4 `GET /api/v1/me/style-profile/suggestions`

*   **Actor:** Khách hàng (User).
*   **Mô tả:** Gợi ý gu từ những món **đang hoạt động** trong tủ (cold start). **Không tự áp dụng**: người
    dùng xác nhận rồi mới gọi `PUT`.
*   **Response `200 OK`:**
    ```json
    {
      "styles": [
        { "value": "minimalist", "itemCount": 12, "ratio": 0.4 },
        { "value": "elegant", "itemCount": 7, "ratio": 0.23 }
      ],
      "sampleSize": 30
    }
    ```
    -   `ratio = itemCount / sampleSize` (làm tròn 2 chữ số).
    -   `sampleSize` = số item active có style không rỗng. Chưa đủ dữ liệu ⇒ `{"styles": [], "sampleSize": 0}`,
        thông báo nằm ở `message` của envelope, không thêm field vào `data`.
*   **Lỗi:** `403` nếu gói không có quyền lợi.

### 4. Ngữ cảnh gu trong pipeline

Khi gói có quyền lợi và hồ sơ đang bật, pipeline gợi ý đọc gu như **ưu tiên mềm**:

*   Vector gu được trộn vào query vector (trọng số 0.30 khi request không có `styleTarget`, 0.10 khi có).
    `styleTarget` của request **luôn thắng** gu lưu sẵn.
*   Prompt thêm `personal_style`, `learned_style`, `avoid_styles`; rule ưu tiên gu chỉ khi chất lượng ứng
    viên tương đương.
*   Không hard-filter: gu không làm mất món hay phá cấu trúc outfit; HSL fallback dùng cùng ngữ cảnh.
*   `enabled = false` trên hồ sơ ⇒ bỏ qua toàn bộ gu, kể cả khi gói còn quyền lợi.

**Bảng mã lỗi mới (033):**

| Mã | HTTP | Khi nào |
|---|---|---|
| `PERSONALIZATION_NOT_AVAILABLE` | 403 | Gói không có quyền cá nhân hoá |
| `PERSONALIZATION_DISABLED` | 403 | Kill switch `personalization.enabled = false` chặn ghi |
| `STYLE_LIMIT_EXCEEDED` | 400 | Vượt số style tối đa |
| `STYLE_VALUE_INVALID` | 400 | Giá trị style > 50 ký tự |
| `RECOMMENDATION_SNAPSHOT_NOT_FOUND` | 404 | Feedback hết hạn/không tồn tại |
| `FEEDBACK_FORBIDDEN` | 403 | Snapshot thuộc user khác |

---

## Flow 2: AI chat stylist

### 1. Tạo chat session
*   **Endpoint:** `POST /api/v1/ai/chat/sessions`
*   **Tác nhân (Actor):** Khách hàng (User).
*   **Đối tượng ảnh hưởng:** Tạo ngữ cảnh hội thoại (conversational context).

### 2. Lấy danh sách chat sessions
*   **Endpoint:** `GET /api/v1/ai/chat/sessions`
*   **Tác nhân (Actor):** Khách hàng (User).
*   **Đối tượng ảnh hưởng:** Đọc ngữ cảnh hội thoại (conversational contexts) của current user.

### 3. Lấy tin nhắn của một session
*   **Endpoint:** `GET /api/v1/ai/chat/sessions/:contextID/messages`
*   **Tác nhân (Actor):** Khách hàng (User).
*   **Đối tượng ảnh hưởng:** Đọc danh sách tin nhắn (messages) của session.

### 4. Đưa session vào lưu trữ (Archive session)
*   **Endpoint:** `PATCH /api/v1/ai/chat/sessions/:contextID/archive`
*   **Tác nhân (Actor):** Khách hàng (User).
*   **Đối tượng ảnh hưởng:** Cập nhật trạng thái session.

### 5. Xóa session
*   **Endpoint:** `DELETE /api/v1/ai/chat/sessions/:contextID`
*   **Tác nhân (Actor):** Khách hàng (User).
*   **Đối tượng ảnh hưởng:** Xóa session và lịch sử liên quan theo use case backend.

### 6. Cập nhật session
*   **Endpoint:** `PATCH /api/v1/ai/chat/sessions/:contextID`
*   **Tác nhân (Actor):** Khách hàng (User).
*   **Đối tượng ảnh hưởng:** Cập nhật metadata session.

### 7. Gửi tin nhắn dạng stream SSE
*   **Endpoint:** `POST /api/v1/ai/chat/sessions/:contextID/messages/stream`
*   **Tác nhân (Actor):** Khách hàng (User).
*   **Đối tượng ảnh hưởng:** Tạo tin nhắn của user (user message) và tin nhắn của trợ lý (assistant message).
*   **Mô tả:** Gửi tin nhắn cho stylist AI và nhận phản hồi (response) dưới dạng Server-Sent Events.
