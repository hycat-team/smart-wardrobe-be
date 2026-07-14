# Đặc Tả Tính Năng: restructure-root-layout

**Feature Branch**: `002-restructure-root-layout`

**Created**: 2026-07-14

**Status**: Draft

**Input**: User description: "Lên kế hoạch để tái cấu trúc thư mục root dự án. Tham khảo repo này https://github.com/golang-standards/project-layout.git. chia lại cấu trúc thư mục hiện tại để dễ dàng tìm kiếm file hơn, không bắt buộc phải đầy đủ các folder như repo mẫu"

## Kịch Bản Người Dùng & Kiểm Thử *(bắt buộc)*

### User Story 1 - Lập trình viên tái cấu trúc các thư mục gốc (Độ ưu tiên: P1)

Là một lập trình viên, tôi muốn có một cấu trúc thư mục dự án chuẩn và sạch sẽ, để tôi có thể dễ dàng tìm kiếm các cấu hình, kịch bản cơ sở dữ liệu, cấu hình triển khai và các gói tiện ích, từ đó bảo trì mã nguồn hiệu quả hơn.

**Lý do ưu tiên**: Dọn dẹp và cấu trúc lại thư mục gốc là tối quan trọng để mở rộng dự án, giúp các lập trình viên và công cụ CI/CD dễ dàng định hướng.

**Kiểm thử độc lập**: Dự án biên dịch thành công bằng lệnh `go build` và vượt qua tất cả các bài kiểm thử bằng lệnh `make test` sau khi cấu trúc lại các thư mục cũng như cập nhật các đường dẫn/import path.

**Kịch bản nghiệm thu**:

1. **Given** cấu trúc dự án hiện tại, **When** thực hiện tổ chức lại, **Then** `nginx/` và `rabbitmq/` được chuyển vào `deployments/`, `migrations/` và `init-db/` được chuyển vào `database/`, và các mã nguồn tiện ích của `pkg/` được chuyển vào `internal/pkg/`.
2. **Given** cấu trúc dự án hiện tại, **When** thực hiện phân tách cấu hình, **Then** các file cấu hình YAML nằm trong thư mục `configs/` ở root, và các file Go đọc cấu hình được đặt trong `internal/config/`.
3. **Given** cấu trúc thư mục mới, **When** chạy ứng dụng, **Then** ứng dụng khởi động thành công mà không gặp lỗi import sai đường dẫn hoặc thiếu cấu hình.

---

### Các Trường Hợp Đặc Biệt (Edge Cases)

- **Cập nhật import path**: Việc di chuyển các file Go trong `pkg/` và `config/` sẽ thay đổi import path của chúng. Tất cả các file phụ thuộc trong toàn bộ mã nguồn phải được cập nhật lại chính xác.
- **Tham chiếu Docker & Triển khai**: Các file Dockerfile, docker-compose.yml, docker-compose.prod.yml, và Makefile có chứa tham chiếu tới `nginx/`, `rabbitmq/`, `init-db/`, `migrations/` hoặc các tệp cấu hình cần được cập nhật đường dẫn mới.

## Yêu Cầu *(bắt buộc)*

### Yêu Cầu Chức Năng

- **FR-001**: Di chuyển các thư mục `nginx/` và `rabbitmq/` vào `deployments/nginx/` và `deployments/rabbitmq/`.
- **FR-002**: Di chuyển các thư mục `migrations/` và `init-db/` vào `database/migrations/` và `database/init/`.
- **FR-003**: Di chuyển mã nguồn Go đọc cấu hình từ `config/` vào `internal/config/`.
- **FR-004**: Di chuyển các file cấu hình YAML từ `config/` vào thư mục `configs/` ở root dự án.
- **FR-005**: Di chuyển toàn bộ thư mục `pkg/` vào `internal/pkg/`.
- **FR-006**: Cập nhật tất cả các câu lệnh import Go trong toàn bộ dự án để phản ánh đúng vị trí thư mục mới.
- **FR-007**: Cập nhật các đường dẫn tham chiếu trong `Dockerfile`, `docker-compose.yml`, `docker-compose.prod.yml`, và `Makefile`.

### Thực Thể Chính

- **Thư mục Deployments**: Thư mục gốc chứa các file cấu hình triển khai và chạy dịch vụ (`deployments/nginx`, `deployments/rabbitmq`).
- **Thư mục Database**: Thư mục gốc chứa trạng thái khởi tạo database và các file migrations (`database/migrations`, `database/init`).
- **Thư mục Configs**: Thư mục gốc chứa các file cấu hình YAML theo từng môi trường (`configs/config.yaml`, `configs/config.production.yaml`).

## Tiêu Chí Thành Công *(bắt buộc)*

### Kết Quả Đo Lường Được

- **SC-001**: Số lượng thư mục ở root được giảm từ 16 xuống gọn gàng hơn, tiệm cận với cấu chuẩn `golang-standards/project-layout`.
- **SC-002**: Thời gian biên dịch mã nguồn (`go build`) hoàn thành thành công dưới 30 giây sau khi tái cấu trúc.
- **SC-003**: Chạy các bài test tự động (`go test ./...`) vượt qua 100% không gặp bất kỳ lỗi import hay thiếu thư viện nào.
- **SC-004**: Lệnh khởi chạy dịch vụ chạy thành công với cấu hình local/production tương ứng.

## Giả Định

- **A-001**: Cho phép thay đổi toàn bộ import path nội bộ vì đây là một dự án Backend độc lập, không cung cấp thư viện cho bên thứ ba import.
- **A-002**: Việc di chuyển thư mục chỉ ảnh hưởng đến đường dẫn biên dịch và đường dẫn file cấu hình, có thể sửa đổi đồng loạt trên toàn dự án.
