# Danh Sách Nhiệm Vụ (Tasks): restructure-root-layout

**Input**: Các tài liệu thiết kế từ `specs/002-project-layout-restructure/`

**Prerequisites**: [plan.md](plan.md) (yêu cầu), [spec.md](spec.md) (yêu cầu)

## Định dạng: `-[ ] [ID] [P?] [Story?] Mô tả`

- **[P]**: Có thể chạy song song (các file khác nhau, không phụ thuộc nhau)
- **[Story]**: Thuộc kịch bản người dùng nào (Ví dụ: US1)
- Các mô tả chứa đường dẫn tương đối (relative paths) tính từ root dự án.

---

## Phase 1: Setup (Shared Infrastructure)

**Mục đích**: Khởi tạo cấu trúc thư mục mới theo phương án thiết kế.

- [ ] T001 Tạo cấu trúc các thư mục mới: `configs/`, `database/init/`, `database/migrations/`, `deployments/nginx/`, `deployments/rabbitmq/`, `internal/config/`, `internal/pkg/`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Mục đích**: Thực hiện di chuyển vật lý các file và thư mục sang vị trí mới trước khi sửa đổi code và cấu hình.

- [ ] T002 Di chuyển các tệp tin cấu hình YAML từ `config/*.yaml` sang `configs/`
- [ ] T003 Di chuyển mã nguồn Go đọc cấu hình từ `config/` sang `internal/config/`
- [ ] T004 Di chuyển mã nguồn tiện ích từ `pkg/` sang `internal/pkg/`
- [ ] T005 Di chuyển kịch bản database từ `init-db/` và `migrations/` sang `database/init/` và `database/migrations/`
- [ ] T006 Di chuyển cấu hình dịch vụ phụ trợ từ `nginx/` và `rabbitmq/` sang `deployments/nginx/` và `deployments/rabbitmq/`
- [ ] T007 Di chuyển các file docker compose từ `docker-compose.yml` và `docker-compose.prod.yml` sang `deployments/`

---

## Phase 3: User Story 1 - Tái cấu trúc thư mục root (Priority: P1) 🎯 MVP

**Mục tiêu**: Cập nhật toàn bộ mã nguồn import, cấu hình Docker/Compose, Makefile, CI/CD workflow và tài liệu tham chiếu để hệ thống hoạt động ổn định với cấu trúc mới.

**Kiểm thử độc lập**: Dự án biên dịch thành công, test chạy qua 100% và container khởi chạy thành công.

### Implementation for User Story 1

- [ ] T008 [US1] Sửa đổi import path trong toàn bộ mã nguồn Go từ `pkg/...` sang `internal/pkg/...` và từ `config/...` sang `internal/config/...`
- [ ] T009 [US1] Cập nhật đường dẫn file cấu hình YAML mặc định trong mã nguồn đọc cấu hình tại `internal/config/`
- [ ] T010 [US1] Cập nhật đường dẫn sao chép file cấu hình YAML trong `Dockerfile`
- [ ] T011 [US1] Cập nhật các đường dẫn volume mount, build context và tệp tin cấu hình trong `deployments/docker-compose.yml` và `deployments/docker-compose.prod.yml`
- [ ] T012 [US1] Cập nhật các lệnh build, docker-compose, và migration trong `Makefile`
- [ ] T013 [US1] Cập nhật các bước đóng gói triển khai (tar) và chạy config trong `.github/workflows/deploy-production.yml`
- [ ] T014 [US1] Cập nhật đường dẫn tham chiếu cũ trong tài liệu `docs/development/local-setup.md`, `docs/system-design/architecture.md`, `docs/system-design/design-pattern.md`

---

## Phase 4: Polish & Verification

**Mục đích**: Xác thực và kiểm thử lại toàn bộ hệ thống sau khi cấu trúc lại.

- [ ] T015 Chạy biên dịch thử nghiệm dự án để kiểm tra import path: `go build -o /tmp/main ./cmd/server`
- [ ] T016 Chạy bộ kiểm thử tự động toàn dự án: `go test ./...`
- [ ] T017 Chạy các công cụ định dạng và phân tích tĩnh: `make check`
- [ ] T018 Khởi chạy Docker Compose và kiểm tra cơ sở dữ liệu khởi tạo, tự động chạy migrations thành công.

---

## Dependencies & Execution Order

- **Setup (Phase 1) & Foundational (Phase 2)**: Chạy tuần tự để chuẩn bị cấu trúc thư mục vật lý mới.
- **User Story 1 (Phase 3)**: Phụ thuộc vào Phase 2. Sau khi file được di chuyển, cập nhật toàn bộ code import và cấu hình hệ thống.
- **Verification (Phase 4)**: Phụ thuộc vào Phase 3. Chỉ thực hiện khi toàn bộ mã nguồn và cấu hình đã được sửa đổi đường dẫn.
