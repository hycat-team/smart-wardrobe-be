# Wardrobe API Specs

Tài liệu này đặc tả các API liên quan đến tủ đồ (wardrobe) trong hệ thống Closy Smart Wardrobe.

## Feature 023 — thay đổi hành vi (trước → sau)

- **Xác nhận món cần rà soát** (đã gỡ): endpoint `POST /api/v1/wardrobe-items/{id}/confirm-review` KHÔNG còn tồn tại (gọi trả `404`); thay bằng `retry-analysis` kèm `categoryId`.
- **Thử lại phân tích**: `POST /api/v1/wardrobe-items/{id}/retry-analysis` — chỉ khi `failed`/`needs_review`; nhận body tùy chọn `{ "categoryId": "<uuid>" }` (bắt buộc khi `needs_review`); chạy chế độ danh mục cố định, giữ danh mục người dùng đã chọn, trả thêm `taskId`.
- **Lọc danh mục/từ khóa rỗng**: `GET /api/v1/wardrobe-items?categorySlug=...` và catalog — danh mục/từ khóa không khớp món nào nay trả **rỗng** (`total=0`) thay vì toàn bộ; không truyền danh mục thì trả toàn bộ (kể cả món chưa gắn fashion item).
- **Realtime**: thêm trạng thái `processing`; replay khi reconnect; realtime là best-effort (nghiệp vụ không bị chặn khi Redis gián đoạn).

---

## Flow 1: Quản lý tủ đồ cá nhân (Customer)

### 1. Lấy chữ ký để tải lên hình ảnh quần áo

- **Endpoint:** `GET /api/v1/wardrobe-items/upload-signature`
- **Tác nhân (Actor):** Khách hàng (User).
- **Đối tượng ảnh hưởng:** Không thay đổi dữ liệu nghiệp vụ.
- **Mô tả:** Lấy Cloudinary signature phục vụ client tải ảnh quần áo trực tiếp lên Cloudinary.

### 2. Lấy danh sách quần áo trong tủ đồ của tôi

- **Endpoint:** `GET /api/v1/me/wardrobe-items`
- **Tác nhân (Actor):** Khách hàng (User).
- **Đối tượng ảnh hưởng:** Đọc danh sách quần áo (wardrobe items) của current user.
- **Mô tả:** Trả về danh sách quần áo của người dùng hiện tại, có tự động áp dụng cơ chế khóa động (lock) dựa trên hạn mức quota của gói dịch vụ đăng ký hiện hành. Trạng thái vật phẩm tham chiếu tại [constants.md](constants.md).

### 3. Lấy số liệu thống kê tủ đồ cá nhân

- **Endpoint:** `GET /api/v1/me/wardrobe-items/stats`
- **Tác nhân (Actor):** Khách hàng (User).
- **Đối tượng ảnh hưởng:** Đọc số liệu thống kê wardrobe/outfit.
- **Mô tả:** Trả về tổng số lượng quần áo đang hoạt động (active wardrobe items) và số lượng bộ phối đồ (outfits) của người dùng.

### 4. Xem chi tiết thông tin một vật phẩm quần áo

- **Endpoint:** `GET /api/v1/wardrobe-items/:id`
- **Tác nhân (Actor):** Khách hàng (User).
- **Đối tượng ảnh hưởng:** Đọc chi tiết một vật phẩm quần áo của current user.
- **Mô tả:** Trả về thông tin chi tiết quần áo, tự động chặn không cho xem nếu vật phẩm đó nằm trong vùng vượt hạn mức bị khóa.

### 5. Khởi tạo tủ đồ nhanh từ danh mục hệ thống

- **Endpoint:** `POST /api/v1/wardrobe-items/catalog-init`
- **Tác nhân (Actor):** Khách hàng (User).
- **Đối tượng ảnh hưởng:** Tạo các bản ghi quần áo mới cho current user.
- **Mô tả:** Sao chép các vật phẩm mẫu từ catalog hệ thống vào tủ đồ cá nhân của người dùng, luồng này được thiết kế không làm tiêu thụ quota AI.

### 6. Tải lên hàng loạt hình ảnh quần áo

- **Endpoint:** `POST /api/v1/wardrobe-items/batch-upload`
- **Tác nhân (Actor):** Khách hàng (User).
- **Đối tượng ảnh hưởng:** Tạo nhiều bản ghi quần áo ở trạng thái xử lý và khởi chạy job phân tích AI (AI analysis).
- **Mô tả:** Tạo nhanh nhiều vật phẩm quần áo dựa trên danh sách các ảnh đã được tải lên Cloudinary trước đó. Trả về mã trạng thái HTTP `202 Accepted` kèm theo một `taskId` đại diện cho phiên upload này để client có thể theo dõi tiến trình qua luồng SSE.
- **Response:**
    - `202 Accepted`:
        ```json
        {
            "success": true,
            "message": "Upload hàng loạt thành công, đang tiến hành phân tích nền",
            "data": {
                "taskId": "a2bc3d4e-5f6g-7h8i-9j0k-1l2m3n4o5p6q"
            }
        }
        ```

### 7. Sao chép nhân bản một vật phẩm quần áo

- **Endpoint:** `POST /api/v1/wardrobe-items/:id/clone`
- **Tác nhân (Actor):** Khách hàng (User).
- **Đối tượng ảnh hưởng:** Tạo bản ghi quần áo mới từ thông tin của vật phẩm có sẵn.
- **Mô tả:** Sao chép nhanh toàn bộ thông tin của một vật phẩm quần áo đã có trong tủ đồ.

### 8. Cập nhật thông tin trang phục

- **Endpoint:** `PATCH /api/v1/wardrobe-items/:id`
- **Tác nhân (Actor):** Khách hàng (User).
- **Đối tượng ảnh hưởng:** Cập nhật giá và danh mục của trang phục.
- **Mô tả:** Cho phép người dùng chỉnh sửa `price` và `categoryId` của trang phục. Các thuộc tính khác (color, style, material...) do AI tự phân tích, người dùng không được sửa.
- **Request Body:**
    ```json
    {
      "price": 150000,
      "categoryId": "uuid-cua-danh-muc"
    }
    ```
- **Response:**
    - `200 OK`: Trả về thông tin trang phục sau cập nhật.

### 9. Yêu cầu phân tích AI lại đối với quần áo bị lỗi

- **Endpoint:** `POST /api/v1/wardrobe-items/:id/retry-analysis`
- **Tác nhân (Actor):** Khách hàng (User).
- **Đối tượng ảnh hưởng:** Tạo mới lại tiến trình phân tích AI (AI analysis job).
- **Mô tả:** Gửi lại yêu cầu AI phân tích hình ảnh đối với các món đồ bị lỗi phân tích hoặc cần xem xét lại.
- **Request Body (tùy chọn):**
    ```json
    {
      "categoryId": "uuid-cua-danh-muc"
    }
    ```
    - `categoryId` **bắt buộc** khi món ở `needs_review` (`uncertain_category`) và phải thuộc nhóm có thể phối đồ; hệ thống chạy lại AI ở chế độ **danh mục cố định** (chỉ sinh metadata, giữ danh mục) nên không tái sinh `needs_review`. Có thể bỏ trống với món `failed` do lỗi tạm thời/hết retry.
- **Lỗi:** thiếu/không hợp lệ `categoryId` khi `needs_review` ⇒ `400`; trạng thái không cho phép ⇒ `400`; ảnh không hợp lệ (đa món/toàn thân) ⇒ `400` (hướng tải ảnh khác).

### 10. Xóa hàng loạt quần áo khỏi tủ đồ

- **Endpoint:** `DELETE /api/v1/wardrobe-items/bulk`
- **Tác nhân (Actor):** Khách hàng (User).
- **Đối tượng ảnh hưởng:** Xóa mềm (soft delete) các bản ghi quần áo được chọn.
- **Mô tả:** Xóa mềm cùng lúc nhiều món đồ khỏi tủ đồ cá nhân.

### 11. Xóa toàn bộ danh sách quần áo vượt hạn mức bị khóa

- **Endpoint:** `DELETE /api/v1/wardrobe-items/locked`
- **Tác nhân (Actor):** Khách hàng (User).
- **Đối tượng ảnh hưởng:** Xóa mềm các bản ghi quần áo đang bị khóa.
- **Mô tả:** Dọn dẹp nhanh các món đồ vượt hạn mức quota đang bị khóa do hạ cấp gói dịch vụ.

### 12. Đăng ký nhận thông báo phân tích thời gian thực qua SSE

- **Endpoint:** `GET /api/v1/wardrobe-items/tasks/:taskId/sse`
- **Tác nhân (Actor):** Khách hàng (User).
- **Đối tượng ảnh hưởng:** Đọc bản tin phân tích thời gian thực.
- **Mô tả:** Mở một kết nối Server-Sent Events (SSE) để nhận cập nhật tiến độ phân tích thời trang tự động của từng sản phẩm trong Task ID. Mỗi khi một sản phẩm được xử lý hoàn tất, bị lỗi hoặc cần xem xét lại, backend sẽ phát một bản tin dạng JSON đến các client đang kết nối.
- **Dữ liệu phản hồi trong Stream (Event Payload):**
    - Mỗi bản tin được phát đi dưới dạng event mặc định (`message`), chứa chuỗi JSON đại diện cho cấu trúc `SSEEventPayload`:
        - `itemId` (string): ID của Wardrobe Item tương ứng.
        - `status` (string): Trạng thái phân tích của item này. Các giá trị hợp lệ bao gồm `completed`, `failed`, `needs_review` (Tham chiếu chi tiết tại [constants.md](constants.md#1-trang-thai-nhiem-vu-phan-tich-analyzetaskstatus)).
        - `total` (int): Tổng số lượng item cần phân tích trong task này.
        - `index` (int): Số thứ tự index của item hiện tại (0-indexed).
        - `data` (object, optional): Chứa dữ liệu WardrobeItem chi tiết khi trạng thái là `completed` hoặc `needs_review`.
        - `error` (string, optional): Thông báo lỗi chi tiết khi trạng thái là `failed`.
    - **Ví dụ Bản tin SSE trả về khi phân tích thành công một trang phục (`completed`):**
        ```json
        {
          "itemId": "e5b8d2a6-b5d7-4b4d-b8d4-f655845c43d9",
          "status": "completed",
          "total": 3,
          "index": 0,
          "data": {
            "id": "e5b8d2a6-b5d7-4b4d-b8d4-f655845c43d9",
            "name": "Áo Thun Unisex Trơn",
            "imageUrl": "https://res.cloudinary.com/closy/image/upload/v12345/shirt.jpg",
            "status": "active",
            "isLocked": false,
            "fashionItem": {
              "id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
              "category": {
                "id": "c2b8d2a6-b5d7-4b4d-b8d4-f655845c43d9",
                "name": "Áo Thun",
                "slug": "ao-thun"
              },
              "imageUrl": "https://res.cloudinary.com/closy/image/upload/v12345/shirt.jpg",
              "color": "Màu đen",
              "colorHex": "#000000",
              "..."
            }
          }
        }
        ```
    - **Ví dụ Bản tin SSE trả về khi phân tích thất bại (`failed`):**
        ```json
        {
            "itemId": "a2f8c1b9-38b4-4b5c-a5b6-7f8a9b0c1d2e",
            "status": "failed",
            "total": 3,
            "index": 1,
            "error": "AI Error: Không tìm thấy trang phục trong hình ảnh hoặc ảnh quá mờ."
        }
        ```
    - **Ví dụ Bản tin SSE trả về khi AI không tự tin phân loại danh mục (`needs_review`):**
        ```json
        {
            "itemId": "d8e9c7a6-28c4-4b5d-a5d6-8e9a0b1c2d3e",
            "status": "needs_review",
            "total": 3,
            "index": 2,
            "data": {
                "id": "d8e9c7a6-28c4-4b5d-a5d6-8e9a0b1c2d3e",
                "name": "Trang phục chưa phân loại",
                "imageUrl": "https://res.cloudinary.com/closy/image/upload/v12345/unknown.jpg",
                "status": "active",
                "isLocked": false
            }
        }
        ```

---

## Flow 2: Danh mục hệ thống và phân loại (System catalog & category)

### 1. Lấy danh sách quần áo mẫu của hệ thống

- **Endpoint:** `GET /api/v1/system-catalog/wardrobe-items`
- **Tác nhân (Actor):** Khách vãng lai / Khách hàng (Public/User).
- **Đối tượng ảnh hưởng:** Đọc bảng dữ liệu catalog hệ thống.
- **Mô tả:** Lấy danh sách catalog quần áo mẫu để người dùng tham khảo, hỗ trợ tìm kiếm nhanh và cơ chế fallback DB.
