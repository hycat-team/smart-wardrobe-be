# Lịch sử thay đổi (Changelog)

## [2026-08-01] - Củng cố ranh giới module (endpoint ownership + auth middleware + swagger per-module)

- **Endpoint vào module**: Mỗi module business (identity, subscription, wardrobe, dashboard, brand, fashion) tự đăng ký endpoint qua `presentation/router/`. `internal/api/routes/` chỉ còn NewEngine + AppRouter.
- **Middleware**: Middleware hạ tầng dùng chung (CORS, timeout, rate-limit, role, swagger lock, error handler) chuyển về `internal/shared/presentation/middleware/`; AuthMiddleware thuộc module identity và được expose qua interface `IAuthMiddleware` trong `presentation/contract/`.
- **Swagger per-module**: Sinh `api/swagger/modules/<tên>/swagger.json` cho 6 module; Swagger UI có dropdown chọn spec theo module.
- **Đổi tag swagger**: Tag `Admin` được tách thành `Admin Identity` / `Admin Wardrobe` / `Admin Brand` theo module sở hữu — thay đổi tài liệu, không phải API contract.

## [2026-06-28] - Refactor tài liệu và cấu trúc Swagger

- **Thay đổi cấu trúc tài liệu**: Tách biệt tài liệu viết tay của con người khỏi Swagger generated files.
- **Chuyển đổi mô hình kinh doanh**: Định vị lại Closy theo hướng B2B2C, đưa các luồng Community và Resale vào archive.
- **Cấu hình Swagger**: Chuyển đầu ra sinh code tự động của Swagger từ thư mục `docs/` sang thư mục mới `api/swagger/`.
- **Bảo mật**: Rà soát và loại bỏ các thông tin nhạy cảm (VPS IP) khỏi tài liệu triển khai.
