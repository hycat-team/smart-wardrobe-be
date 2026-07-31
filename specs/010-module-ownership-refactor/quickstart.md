# Quickstart — Củng cố Ranh Giới Module

**Branch**: `010-module-ownership-refactor` | **Date**: 2026-08-01

Hướng dẫn validate end-to-end cho từng đợt. Chi tiết kỹ thuật xem [plan.md](plan.md), contracts và data-model.

## Prerequisites

- Go v1.25+, Docker (nếu cần chạy stack), `make` (Windows PowerShell/Unix — Makefile đã hỗ trợ).
- Cài tools: `make install-tools` (Wire, Swag).
- DB chạy: `make docker-dev-up` (nếu cần chạy server thật; route-registry test KHÔNG cần DB).

## Đợt 0 — Tách middleware

### Validate
```bash
# 1. Route-registry baseline: TRƯỚC khi refactor, sinh golden từ code hiện tại
go test ./internal/api/routes/ -run TestRoutesBaseline -update

# 2. Refactor middleware xong, kiểm tra
make build          # biên dịch
go test ./...       # toàn bộ test pass (route-registry, errorutils...)
```

### Kiểm tra thủ công (server thật)
```bash
make dev   # hoặc make run
```
- Gọi endpoint public (vd `GET /api/v1/health`) → 200.
- Gọi endpoint private không JWT → 401 message tiếng Việt.
- Gọi với JWT hợp lệ → 200.
- CORS header xuất hiện; request quá nhanh → 429; path không tồn tại → 404 JSON.

### Expected
- Không package nào trong `internal/shared/presentation/middleware` import `internal/modules/`.
- `internal/api/middleware` đã xóa.
- Route-registry test pass (baseline không đổi).

## Đợt 1 — Module tự định nghĩa endpoint

### Validate
```bash
make generate    # Wire regenerate (wire_gen.go)
make build
make test
```

### Kiểm tra thủ công
```bash
make run
```
- `GET /api/v1/health` → 200.
- Một endpoint đại diện mỗi module → phản hồi đúng (xem contracts/route-registry.md).
- Không panic route conflict khi khởi động.

### Expected
- Route-registry test pass (path/method giữ nguyên 100%).
- `internal/api/routes/` chỉ còn `router.go` + `provider.go`; 10 folder cũ đã xóa.
- Không router module nào import handler module khác (ngoại trừ `identity_contract.IAuthMiddleware`).

## Đợt 2 — Swagger per-module

### Validate
```bash
make generate    # wire + swag init + swagger-split
make generate-check   # diff master + modules
```

### Kiểm tra thủ công
- Mở `http://localhost:<port>/swagger`.
- Dropdown (cạnh version selector) hiển thị: `v1 - Tất cả`, `identity`, `subscription`, `wardrobe`, `dashboard`, `brand`, `fashion`.
- Chọn từng module → chỉ hiển thị endpoint module đó.
- Chọn `v1 - Tất cả` → đủ 130 paths như trước.

### Expected
- `api/swagger/modules/` có 6 file `swagger.json`.
- Tổng paths 6 spec == paths master.
- `make generate-check` pass (không diff bất ngờ).

## Gate cuối

```bash
make check   # fmt-check, tidy-check, generate-check, vet, test, build
```

Tất cả pass → feature hoàn tất, sẵn sàng tạo tasks (`/speckit-tasks`).
