# Thiết kế Cơ sở Dữ liệu (Database Design)

Hệ thống sử dụng cơ sở dữ liệu quan hệ PostgreSQL làm nguồn dữ liệu tin cậy duy nhất (Single Source of Truth), tích hợp với GORM để quản lý thực thể.

## 1. Các bảng dữ liệu chính (Core Tables)

- `users`: Lưu trữ thông tin tài khoản người dùng B2C.
- `wardrobe_items`: Quản lý quần áo số hóa và đường dẫn ảnh.
- `outfits`: Lưu trữ các bộ phối đồ đã lưu.
- `brands`: Thông tin các thương hiệu đối tác.
- `brand_members`: Quản lý vai trò nhân viên thuộc thương hiệu.
- `loyalty_points`: Điểm tích lũy của người dùng tại từng thương hiệu.
- `campaigns`: Các sự kiện, chiến dịch tiếp thị của nhãn hàng.
- `digital_samples`: Lưu trữ thiết kế mẫu thử trong Sample Lab.
- `sample_votes`: Lịch sử bình chọn của người dùng cho mẫu thiết kế.
