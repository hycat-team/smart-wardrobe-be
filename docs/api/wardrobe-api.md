# Wardrobe & Outfit API Specs

Tài liệu này bổ sung đặc tả các API liên quan đến tủ đồ (wardrobe) và trang phục (outfit) đang được triển khai (implemented) trong codebase cùng các thay đổi về mô hình sau đợt B2B2C rebuild.

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
- **Mô tả:** Trả về danh sách quần áo của người dùng hiện tại, có tự động áp dụng cơ chế khóa động (lock) dựa trên hạn mức quota của gói dịch vụ đăng ký hiện hành. Trạng thái vật phẩm tham chiếu tại [constants/wardrobe.md](constants/wardrobe.md).

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

### 8. Phân loại thuộc tính thời trang thủ công cho quần áo

- **Endpoint:** `PUT /api/v1/wardrobe-items/:id/manual-classify`
- **Tác nhân (Actor):** Khách hàng (User).
- **Đối tượng ảnh hưởng:** Cập nhật thông tin quần áo và dữ liệu vector embedding liên quan.
- **Mô tả:** Người dùng tự điền thông tin thuộc tính thời trang cho quần áo bị phân tích lỗi hoặc cần duyệt lại thủ công.

### 9. Yêu cầu phân tích AI lại đối với quần áo bị lỗi

- **Endpoint:** `POST /api/v1/wardrobe-items/:id/retry-analysis`
- **Tác nhân (Actor):** Khách hàng (User).
- **Đối tượng ảnh hưởng:** Tạo mới lại tiến trình phân tích AI (AI analysis job).
- **Mô tả:** Gửi lại yêu cầu AI phân tích hình ảnh đối với các món đồ bị lỗi phân tích hoặc cần xem xét lại.

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
        - `status` (string): Trạng thái phân tích của item này. Các giá trị hợp lệ bao gồm `completed`, `failed`, `needs_review` (Tham chiếu chi tiết tại [constants/task.md](constants/task.md#1-trang-thai-nhiem-vu-phan-tich-analyzetaskstatus)).
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

### 2. Lấy danh sách các danh mục thời trang

- **Endpoint:** `GET /api/v1/categories`
- **Tác nhân (Actor):** Khách vãng lai / Khách hàng (Public/User).
- **Đối tượng ảnh hưởng:** Đọc bảng danh mục thời trang `categories`.
- **Mô tả:** Trả về cấu trúc phân loại danh mục (category taxonomy) đang sử dụng trong hệ thống tủ đồ Closy.

### 3. Quản trị viên quản lý danh mục quần áo mẫu hệ thống

- **Endpoints:**
    - `GET /api/v1/admin/wardrobe-items`
    - `PUT /api/v1/admin/wardrobe-items/:id`
    - `DELETE /api/v1/admin/wardrobe-items/:id`
    - `GET /api/v1/admin/wardrobe-items/upload-signature`
    - `POST /api/v1/admin/wardrobe-items/batch-upload`
- **Tác nhân (Actor):** Quản trị viên (Admin).
- **Mô tả:** Thực hiện quản lý thêm, sửa, xóa catalog quần áo mẫu của hệ thống.

### 4. Quản trị viên quản lý danh mục phân loại (Categories)

- **Endpoints:**
    - `GET /api/v1/admin/categories`
    - `GET /api/v1/admin/categories/:id`
    - `POST /api/v1/admin/categories`
    - `PUT /api/v1/admin/categories/:id`
    - `DELETE /api/v1/admin/categories/:id`
- **Tác nhân (Actor):** Quản trị viên (Admin).
- **Mô tả:** Quản trị cấu trúc cây phân loại danh mục thời trang.

---

## Flow 3: Tự thiết kế bộ phối đồ (Outfit)

### 1. Lấy chữ ký tải lên ảnh bìa bộ phối đồ

- **Endpoint:** `GET /api/v1/outfits/upload-signature`
- **Tác nhân (Actor):** Khách hàng (User).
- **Đối tượng ảnh hưởng:** Không thay đổi dữ liệu nghiệp vụ.
- **Mô tả:** Lấy Cloudinary signature phục vụ client tải ảnh chụp/render cover của outfit lên Cloudinary.

### 2. Lưu bộ phối đồ (outfit) tự thiết kế mới

- **Endpoint:** `POST /api/v1/outfits`
- **Tác nhân (Actor):** Khách hàng (User).
- **Đối tượng ảnh hưởng:** Tạo bản ghi outfit và danh sách chi tiết các item thuộc bộ phối đồ (`outfit_items`).
- **Mô tả:** Lưu trữ thông tin bộ trang phục do người dùng tự phối ghép bao gồm danh sách các item thời trang, tọa độ hiển thị 2D trên canvas và thứ tự hiển thị layer của từng item.

### 3. Cập nhật thông tin bộ phối đồ

- **Endpoint:** `PUT /api/v1/outfits/:id`
- **Tác nhân (Actor):** Khách hàng (User).
- **Đối tượng ảnh hưởng:** Cập nhật thông tin outfit của current user.

### 4. Lấy danh sách các bộ phối đồ của tôi

- **Endpoint:** `GET /api/v1/me/outfits`
- **Tác nhân (Actor):** Khách hàng (User).
- **Đối tượng ảnh hưởng:** Đọc danh sách outfits của current user.

### 5. Xem thông tin chi tiết một bộ phối đồ

- **Endpoint:** `GET /api/v1/outfits/:id`
- **Tác nhân (Actor):** Khách hàng (User).
- **Đối tượng ảnh hưởng:** Đọc chi tiết một bản ghi outfit của current user.
- **Mô tả:** Trả về thông tin chi tiết của outfit và danh sách các item thời trang đi kèm để hiển thị vẽ lại trên canvas của app.

### 6. Xóa bộ phối đồ khỏi danh sách

- **Endpoint:** `DELETE /api/v1/outfits/:id`
- **Tác nhân (Actor):** Khách hàng (User).
- **Đối tượng ảnh hưởng:** Xóa mềm bản ghi outfit của current user.

---

## B2B2C Model Notes

- Món đồ cá nhân trong tủ đồ (`wardrobe_items`) và sản phẩm của nhãn hàng (`brand_items`) đều sử dụng chung một cơ chế tham chiếu thông tin dữ liệu thời trang qua bảng trung gian `fashion_items`.
- Danh sách chi tiết bộ phối đồ (`outfit_items`) sử dụng khóa ngoại `fashion_item_id` kết hợp với trường bối cảnh `item_context` để phân biệt rõ món đồ đó được lấy từ tủ đồ cá nhân hay từ Digital Sample Lab của nhãn hàng đối tác. Giá trị trường bối cảnh `item_context` tham chiếu chi tiết tại [constants/wardrobe.md](constants/wardrobe.md).
- Khi lưu trữ hoặc xem chi tiết bộ phối đồ có chứa sản phẩm nhãn hàng (brand items), hệ thống backend bắt buộc phải xác thực sản phẩm đó thuộc nhãn hàng hợp lệ và tài khoản người dùng có quyền tiếp cận theo đúng các chính sách loyalty/benefit đang áp dụng.
