# Chiến lược viết test (Testing Strategy)

Đảm bảo độ tin cậy của mã nguồn khi nâng cấp và bảo trì hệ thống.

## 1. Unit Testing
*   Viết test độc lập cho từng module tại `usecase` hoặc `repository`.
*   Mock các kết nối ngoài (Database, Cache, Third-party AI) sử dụng GoMock hoặc Testify.

## 2. Cách chạy bộ kiểm thử
Chạy toàn bộ các bài kiểm tra tự động trên môi trường local:
```bash
make test
```
Đảm bảo tất cả test case đều pass trước khi gửi Pull Request.
