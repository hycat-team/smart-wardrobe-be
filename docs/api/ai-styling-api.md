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
*   **Response:**
    *   `200 OK`: Trả về kết quả gợi ý outfit, danh sách items và giải thích của AI.

### Quy tắc B2B2C
*   `includeBrandItems = true` không có nghĩa bắt buộc phải dùng brand item; AI chỉ nên dùng nếu phù hợp với outfit.
*   Fashion module gọi Brand contract để lấy danh sách brand items được phép, không tự đọc chi tiết loyalty/benefit.
*   Brand item candidate phải thỏa mãn điều kiện brand active, item active, và user có quyền truy cập hợp lệ (access) nếu item yêu cầu đặc quyền (benefit). Các benefit/feature code tham chiếu [constants/brand.md](constants/brand.md).
*   AI output không được tự ý inject brand item không nằm trong candidate list mà backend đã cung cấp.
*   Item trong outfit response cần giữ `itemContext = brand_item`; giá trị tham chiếu [constants/wardrobe.md](constants/wardrobe.md).

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
