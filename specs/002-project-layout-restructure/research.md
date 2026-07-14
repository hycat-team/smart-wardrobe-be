# Nghiên Cứu Tái Cấu Trúc Layout Dự Án

Tài liệu nghiên cứu này phân tích việc tái cấu trúc thư mục root dựa trên tiêu chuẩn thiết kế cấu trúc thư mục của Go (Go Standard Project Layout).

## 1. Phân Tích Cấu Trúc Hiện Tại
Hiện tại, dự án có quá nhiều thư mục ở root level (16 thư mục), gây nhiễu và khó tìm kiếm:
- `nginx/`, `rabbitmq/` chứa các cấu hình hệ thống bổ trợ.
- `migrations/`, `init-db/` chứa các kịch bản database.
- `pkg/` chứa logger và utils.
- `config/` chứa cả code Go đọc cấu hình lẫn các file YAML cấu hình.

## 2. Giải Pháp Tái Cấu Trúc Đề Xuất
Áp dụng các đề xuất từ `golang-standards/project-layout`:
- **`/deployments`**: Nơi lưu trữ các cấu hình hệ thống/triển khai. Chúng tôi gom `nginx/` và `rabbitmq/` vào đây.
- **`/database`**: Tên thư mục trực quan chứa các kịch bản khởi tạo (`init/`) và tệp migration (`migrations/`).
- **`/configs`**: Lưu trữ các file cấu hình YAML thuần túy.
- **`/internal`**: Chứa toàn bộ mã nguồn private của ứng dụng.
  - Phân tách code Go đọc cấu hình từ `config/` vào `internal/config/`.
  - Di chuyển thư mục `pkg/` sang `internal/pkg/` do dự án này không chia sẻ mã nguồn tiện ích ra ngoài.

## 3. Rủi Ro & Phương Án Khắc Phục
- **Lỗi Import Path**: Có khoảng hàng chục file Go đang import từ `pkg/utils`, `pkg/logger`, `config`. Ta sẽ dùng công cụ tìm kiếm và thay thế (grep/sed hoặc các tác vụ refactor của Editor) để cập nhật đồng loạt các import path này sang `github.com/.../internal/pkg/...` hoặc `github.com/.../internal/config/...`.
- **Lỗi đường dẫn trong Docker/Compose/Makefile**: Cập nhật lại toàn bộ đường dẫn trong Dockerfile và docker-compose.
