# Category API Specs

Tài liệu này đặc tả các API liên quan đến danh mục thời trang (categories) trong hệ thống Closy Smart Wardrobe.

---

## Danh mục thời trang (Categories)

### 1. Lấy danh sách các danh mục thời trang

- **Endpoint:** `GET /api/v1/categories`
- **Tác nhân (Actor):** Khách vãng lai / Khách hàng (Public/User).
- **Đối tượng ảnh hưởng:** Đọc bảng danh mục thời trang `categories`.
- **Mô tả:** Trả về cấu trúc phân loại danh mục (category taxonomy) đang sử dụng trong hệ thống tủ đồ Closy.
