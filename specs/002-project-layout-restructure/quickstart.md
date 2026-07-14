# Hướng Dẫn Xác Thực Nhanh (Quickstart)

Tài liệu này hướng dẫn chạy xác thực việc tái cấu trúc thư mục để đảm bảo hệ thống vẫn biên dịch và hoạt động bình thường.

## Các Bước Kiểm Tra Sau Khi Tái Cấu Trúc

### 1. Kiểm tra biên dịch (Compilation)
Đảm bảo tất cả import path mới đã được cập nhật thành công:
```bash
go build -o /tmp/main ./cmd/server
```
Lệnh này phải hoàn thành mà không gặp lỗi biên dịch.

### 2. Kiểm tra Test tự động (Testing)
Chạy bộ kiểm thử tự động của toàn bộ dự án:
```bash
go test ./...
```
Tất cả các test case phải pass 100%.

### 3. Kiểm tra static check (Static Analysis)
Chạy các công cụ định dạng và phân tích tĩnh:
```bash
make check
```
Không được có lỗi liên quan đến đường dẫn tệp tin cấu hình hoặc import.

### 4. Chạy ứng dụng locally bằng Docker Compose
Khởi động cụm dịch vụ lên:
```bash
docker compose down -v
docker compose up -d --build
```
Kiểm tra logs của container backend:
```bash
docker compose logs -f backend
```
Backend phải chạy thành công, kết nối được database, tự động migrate cơ sở dữ liệu và sẵn sàng nhận request.
