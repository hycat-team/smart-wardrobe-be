# Admin Wardrobe API Specs

Tài liệu này đặc tả các API quản trị liên quan đến tủ đồ và danh mục trong hệ thống Closy Smart Wardrobe.

---

## Flow 2 part 2: Quản trị viên quản lý danh mục quần áo mẫu hệ thống

- **Endpoints:**
    - `GET /api/v1/admin/wardrobe-items`
    - `PUT /api/v1/admin/wardrobe-items/:id`
    - `DELETE /api/v1/admin/wardrobe-items/:id`
    - `GET /api/v1/admin/wardrobe-items/upload-signature`
    - `POST /api/v1/admin/wardrobe-items/batch-upload`
- **Tác nhân (Actor):** Quản trị viên (Admin).
- **Mô tả:** Thực hiện quản lý thêm, sửa, xóa catalog quần áo mẫu của hệ thống.
- **Lưu ý về upload:** `upload-signature` trả `remainingSlots`/`maxItems`/`currentItems` bằng `-1`
  (admin không bị giới hạn quota). `publicId` **không có** trong response; client không được gửi
  `public_id` lên Cloudinary. Xem luồng 2 bước tại
  [wardrobe-api.md mục 6](wardrobe-api.md#6-tải-lên-hàng-loạt-hình-ảnh-quần-áo).

---

## Flow 2 part 3: Quản trị viên quản lý danh mục phân loại (Categories)

- **Endpoints:**
    - `GET /api/v1/admin/categories`
    - `GET /api/v1/admin/categories/:id`
    - `POST /api/v1/admin/categories`
    - `PUT /api/v1/admin/categories/:id`
    - `DELETE /api/v1/admin/categories/:id`
- **Tác nhân (Actor):** Quản trị viên (Admin).
- **Mô tả:** Quản trị cấu trúc cây phân loại danh mục thời trang.
