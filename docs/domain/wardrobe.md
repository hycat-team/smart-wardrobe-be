# Nghiệp vụ Tủ đồ số (Wardrobe Domain)

Nghiệp vụ quản lý tủ đồ số là cốt lõi của trải nghiệm B2C, cho phép ảo hóa tủ đồ thực tế.

## 1. Wardrobe Item (Trang phục)
Mỗi trang phục trong hệ thống được quản lý bởi các thuộc tính:
*   **ID**: Mã định danh duy nhất.
*   **Media**: URL ảnh gốc đã tải lên và ảnh đã qua xử lý tách nền.
*   **Category (Danh mục)**: Áo (Top), Quần/Váy (Bottom), Đầm (Dress), Giày (Shoes), Phụ kiện (Accessory).
*   **Properties (Đặc tính)**: Màu sắc (Color), Chất liệu (Material), Mùa phù hợp (Season), Phong cách (Style).
*   **Decay State**: Trạng thái cũ/mới của trang phục theo thời gian sử dụng (Garment Lifecycle Decay).

## 2. Closet Initialization Catalog
Quy trình khởi tạo tủ đồ nhanh dành cho người dùng mới bằng cách chọn các trang phục cơ bản từ Catalog hệ thống thay vì phải chụp và upload từng món đồ của mình.
