# Mô Hình Dữ Liệu - Tái Cấu Trúc Layout Dự Án

Việc tái cấu trúc thư mục root không làm thay đổi bất kỳ mô hình dữ liệu (Database Schema, Tables) hay các struct Entity trong code.

## Chi Tiết
- **Cơ sở dữ liệu**: Không có bảng mới được tạo hoặc xóa.
- **Thư mục lưu trữ**: Các file Goose migration được chuyển từ `/migrations` sang `/database/migrations`.
- **Kịch bản SQL khởi tạo**: Các file SQL khởi tạo được chuyển từ `/init-db` sang `/database/init`.
