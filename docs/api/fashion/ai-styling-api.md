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
      "remainingQuota": 12
    }
    ```
    **Ghi chú response:**
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
