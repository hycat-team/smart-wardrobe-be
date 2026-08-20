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
