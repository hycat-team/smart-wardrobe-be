# Kế Hoạch Triển Khai: restructure-root-layout

**Branch**: `002-project-layout-restructure` | **Date**: 2026-07-14 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `spec.md`

## Tóm tắt (Summary)
Tái cấu trúc thư mục root của dự án SmartWardrobe Backend tiệm cận với tiêu chuẩn `golang-standards/project-layout`. Toàn bộ cấu hình dịch vụ, cơ sở dữ liệu, file cấu hình YAML, cấu hình Go và các package tiện ích sẽ được phân bổ lại để làm sạch thư mục gốc và tăng khả năng tìm kiếm file.

## Kịch Bản Kỹ Thuật (Technical Context)

- **Language/Version**: Golang 1.25.x
- **Primary Dependencies**: Gin Web Framework, GORM, Goose, Wire, RabbitMQ client, Redis client
- **Storage**: PostgreSQL (with pgvector), Redis
- **Testing**: `go test ./...`
- **Target Platform**: Linux VPS, Docker container
- **Project Type**: Web Service (Backend REST API)
- **Performance Goals**: Không thay đổi (biên dịch nhanh, hoạt động ổn định)
- **Constraints**: Phải đảm bảo toàn bộ mã nguồn import chính xác và CI/CD workflow trên Github Actions hoạt động bình thường sau tái cấu trúc.
- **Scale/Scope**: Tái cấu trúc cấu trúc thư mục root mà không làm ảnh hưởng đến logic nghiệp vụ.

## Kiểm Tra Hiến Pháp (Constitution Check)

- **Hiến Pháp**: Đã đối chiếu với `SmartWardrobe Backend Constitution v1.1.0`.
- **Yêu cầu database**: Mọi thay đổi schema phải thông qua Goose SQL migrations. Việc tái cấu trúc này di chuyển thư mục chứa tệp migration sang `database/migrations/`, cần cập nhật đường dẫn tương ứng trong Makefile để các lệnh `make migration-*` tiếp tục hoạt động chính xác.
- **Ngôn ngữ**: Các file cấu hình yaml mới nằm ở `configs/`, file Go parser mới nằm ở `internal/config/`. Tài liệu kỹ thuật dự án nằm trong `specs/` phải dùng tiếng Việt.

## Project Structure

### Documentation (this feature)

```text
specs/002-project-layout-restructure/
├── spec.md              # Tài liệu đặc tả yêu cầu
├── plan.md              # File kế hoạch này
├── research.md          # Kết quả nghiên cứu cấu trúc thư mục (Phase 0)
├── data-model.md        # Thiết kế mô hình dữ liệu (Không có thay đổi - Phase 1)
├── quickstart.md        # Hướng dẫn kiểm thử và chạy nghiệm thu (Phase 1)
└── checklists/
    └── requirements.md  # Checklist chất lượng yêu cầu
```

### Thay Đổi Cấu Trúc Thư Mục (Source Code)

```text
Cấu trúc mới sau khi tái cấu trúc:
smart-wardrobe-be/
├── configs/                          # [NEW] Chứa các tệp YAML cấu hình
│   ├── config.yaml
│   └── config.production.yaml
├── database/                         # [NEW] Chứa tài nguyên database
│   ├── init/                         # (Di chuyển từ init-db/)
│   └── migrations/                   # (Di chuyển từ migrations/)
├── deployments/                      # [NEW] Chứa cấu hình dịch vụ phụ trợ & docker-compose
│   ├── nginx/                        # (Di chuyển từ nginx/)
│   ├── rabbitmq/                     # (Di chuyển từ rabbitmq/)
│   ├── docker-compose.yml            # (Di chuyển từ root)
│   └── docker-compose.prod.yml       # (Di chuyển từ root)
├── internal/
│   ├── config/                       # [NEW] (Di chuyển parser code từ config/)
│   ├── pkg/                          # [NEW] (Di chuyển từ pkg/)
│   └── ...                           # (Các module nghiệp vụ hiện tại)
├── cmd/
│   └── server/                       # Main entrypoint
├── docs/                             # Tài liệu dự án (Cập nhật tham chiếu đường dẫn)
├── .github/                          # Github Actions CI/CD workflows (Cập nhật tham chiếu)
├── Dockerfile                        # Cập nhật đường dẫn copy config
└── Makefile                          # Cập nhật đường dẫn các lệnh docker-compose, migration và check
```

## Proposed Changes

### [Component: Project Layout Restructuring]

#### [NEW] [configs/](../../configs/)
#### [NEW] [database/](../../database/)
#### [NEW] [deployments/](../../deployments/)
#### [NEW] [internal/config/](../../internal/config/)
#### [NEW] [internal/pkg/](../../internal/pkg/)

#### [MODIFY] [Dockerfile](../../Dockerfile)
#### [MODIFY] [Makefile](../../Makefile)
#### [MODIFY] [.github/workflows/deploy-production.yml](../../.github/workflows/deploy-production.yml)
#### [MODIFY] [docs/development/local-setup.md](../../docs/development/local-setup.md)
#### [MODIFY] [docs/system-design/architecture.md](../../docs/system-design/architecture.md)
#### [MODIFY] [docs/system-design/design-pattern.md](../../docs/system-design/design-pattern.md)

#### [DELETE] [pkg/](../../pkg/)
#### [DELETE] [config/](../../config/)
#### [DELETE] [nginx/](../../nginx/)
#### [DELETE] [rabbitmq/](../../rabbitmq/)
#### [DELETE] [migrations/](../../migrations/)
#### [DELETE] [init-db/](../../init-db/)
#### [DELETE] [docker-compose.yml](../../docker-compose.yml)
#### [DELETE] [docker-compose.prod.yml](../../docker-compose.prod.yml)

---

## Verification Plan

### Automated Tests
- Chạy biên dịch toàn bộ dự án: `go build -o /tmp/main ./cmd/server`
- Chạy kiểm thử: `go test ./...`
- Chạy các bước xác thực tĩnh của dự án: `make check`

### Manual Verification
- Chạy ứng dụng locally bằng docker-compose thông qua Makefile: `make dev-up` (hoặc `docker compose -f deployments/docker-compose.yml up -d`)
- Kiểm tra kết nối cơ sở dữ liệu và các migrations chạy tự động khi start.
- Xác nhận các file cấu hình YAML được đọc chính xác từ thư mục mới `configs/`.
