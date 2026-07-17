# Hướng dẫn Kiểm thử và Xác thực: Chuẩn hoá các Hằng số Kỹ thuật của AI Executor

Tài liệu này hướng dẫn cách thức kiểm thử, xác thực việc tái cấu trúc các hằng số và kiểu dữ liệu trong module `aiexecutor`.

## Các Bước Chuẩn Bị & Chạy Kiểm Thử

### 1. Dọn dẹp các import không hợp lệ và biên dịch dự án
Sau khi áp dụng các thay đổi trong kế hoạch triển khai, chạy lệnh build để đảm bảo trình biên dịch kiểm lỗi type-safe thành công và không bị sót kiểu cũ:
```bash
go build ./...
```

### 2. Thực thi kiểm thử tự động (Unit / Integration Tests)
Chạy bộ test hiện tại của module `aiexecutor` để xác minh chức năng nghiệp vụ của orchestrator, pricing configs, và usage ledger vẫn chạy đúng:
```bash
go test -v ./internal/modules/aiexecutor/...
```

### 3. Xác thực bằng tay trên môi trường local
- Chạy hệ thống ở chế độ phát triển (development):
  ```bash
  make dev
  ```
- Tiến hành thực hiện một luồng AI call bất kỳ (ví dụ: Outfit recommendation hoặc Chat) để đảm bảo orchestrator vẫn có thể parse chính xác các thông tin cấu hình từ cơ sở dữ liệu (tương thích ngược định dạng text trong GORM).
- Kiểm tra log hệ thống để đảm bảo việc định tuyến (Route: Paid, Free, Local) hoạt động đúng như mong đợi.
