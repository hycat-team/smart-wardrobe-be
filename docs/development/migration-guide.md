# Hướng dẫn tạo và chạy Migration (Migration Guide)

Hệ thống sử dụng **Goose** làm công cụ quản lý các lượt cập nhật cơ sở dữ liệu.

## 1. Nguyên tắc cốt lõi
*   Tuyệt đối không được chỉnh sửa trực tiếp các script cơ sở dữ liệu đã có trong thư mục `/init-db`.
*   Mọi thay đổi lược đồ database phải được thực hiện thông qua file migration sql mới được tạo bằng Goose.

## 2. Tạo một file migration mới
Chạy lệnh sau trên terminal để sinh file migration mới:
```bash
make migration-create name=ten_file_migration
```
Thao tác này sẽ tạo một file `.sql` mới trong thư mục `/migrations` có định dạng thời gian và tên bạn vừa nhập.

## 3. Thực thi migration
Để chạy tất cả các migration chưa được thực thi lên cơ sở dữ liệu local:
```bash
make migration-up
```

Để rollback migration gần nhất:
```bash
make migration-down
```
