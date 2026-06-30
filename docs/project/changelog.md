# Lịch sử thay đổi (Changelog)

## [2026-06-28] - Refactor tài liệu và cấu trúc Swagger

- **Thay đổi cấu trúc tài liệu**: Tách biệt tài liệu viết tay của con người khỏi Swagger generated files.
- **Chuyển đổi mô hình kinh doanh**: Định vị lại Closy theo hướng B2B2C, đưa các luồng Community và Resale vào archive.
- **Cấu hình Swagger**: Chuyển đầu ra sinh code tự động của Swagger từ thư mục `docs/` sang thư mục mới `api/swagger/`.
- **Bảo mật**: Rà soát và loại bỏ các thông tin nhạy cảm (VPS IP) khỏi tài liệu triển khai.
