# Implementation Plan: Củng cố Ranh Giới Module — Endpoint Ownership, Auth Middleware & Swagger Per-Module

**Branch**: `010-module-ownership-refactor` | **Date**: 2026-08-01 | **Spec**: [spec.md](specs/010-module-ownership-refactor/spec.md)

**Input**: Feature specification từ `/specs/010-module-ownership-refactor/spec.md`

## Summary

Refactor modular monolith để mỗi module tự sở hữu surface area (endpoint), auth middleware thuộc về module identity, và swagger được sinh/tách theo module. Chia 3 đợt:

1. **Đợt 0 — Tách middleware**: 6 middleware hạ tầng (cors, timeout, ratelimit, role, swagger_lock, error_handler) chuyển về `internal/shared/presentation/middleware/`; `auth.go` chuyển về `internal/modules/identity/presentation/middleware/`, expose qua interface `IAuthMiddleware` trong `identity/presentation/contract/`.
2. **Đợt 1 — Module tự định nghĩa endpoint**: Tạo `presentation/router/` cho 6 module (identity, subscription, wardrobe, dashboard, brand, fashion); phân tách các route cross-module đang trộn lẫn; `internal/api/routes/` chỉ còn NewEngine + AppRouter.
3. **Đợt 2 — Swagger per-module**: Realign `@Tags` theo module (mỗi tag thuộc đúng 1 module), script split master spec thành `api/swagger/modules/<tên>.json`, dropdown trong Swagger UI.

Không tách database schema. Không breaking change (được bảo vệ bằng automated route-registry test).

## Technical Context

**Language/Version**: Go (Golang v1.25+)

**Primary Dependencies**: Gin (`github.com/gin-gonic/gin`), Google Wire (`github.com/google/wire`), Swag (`github.com/swaggo/swag`), Goose (migrations)

**Storage**: Không đổi — PostgreSQL dùng chung, không tách schema

**Testing**: `go test` qua `make test`; `make check` (fmt-check, tidy-check, generate-check, vet, test, build). Thêm route-registry snapshot test.

**Target Platform**: Linux server (production), Windows/Linux (dev) — Makefile đã hỗ trợ cả 2 (Windows PowerShell + Unix branches)

**Project Type**: Web service (backend Go, modular monolith, single binary)

**Performance Goals**: Không đổi — refactor cấu trúc, không tác động hiệu năng

**Constraints**:
- Không breaking change với client (130 paths giữ nguyên)
- Không tạo import cycle (Go build fail nếu có)
- `shared` không được phụ thuộc business module
- Module không import handler/implementation module khác

**Scale/Scope**: 6 module business, ~130 API paths, 8 file middleware, 10 folder route cũ, ~30+ file handler annotation

## Constitution Check

*GATE: Phải pass trước Phase 0 research. Re-check sau Phase 1 design.*

Đối chiếu với `.specify/memory/constitution.md` v1.4.0:

- **I. Pragmatic Modular Monolith** ✅ — Feature củng cố đúng nguyên tắc này: module tự quản endpoint, middleware tách theo tầng, shared infra dùng chung qua DI.
- **II. Strict Module Boundaries & DTO Contracts** ✅ — Đưa endpoint về module, route module không import handler module khác; auth dùng qua contract `IAuthMiddleware`. Không vi phạm.
- **IV. Clean Architecture & Strict Layer Dependency** ✅ — Middleware phi-auth thuộc `shared/presentation` (tầng presentation của shared, không phụ thuộc module); `auth.go` thuộc `identity/presentation` (đúng layer presentation của identity). Router nằm trong `presentation/router/` — presentation layer. Không layer nào skip.
- **V. Standardized Transaction & Error Handling** ✅ — Giữ nguyên `apperror` + global error middleware (chỉ đổi vị trí sang shared).
- **VI. AI Pipeline** ✅ — Không đụng.
- **Tech Stack** ✅ — Go + Wire + Gin + Swag, không đổi.
- **Language Standards** ✅ — Swagger/validation message tiếng Việt giữ nguyên; comment/code tiếng Anh.
- **Access Control** ✅ — RBAC giữ nguyên (auth middleware + RolesAuthorize giữ nguyên logic).

**Không có vi phạm** → Gate pass, không cần Complexity Tracking.

**Lưu ý cần cập nhật Constitution (đợt sau, ngoài phạm vi code)**: Kiến trúc hiện tại mô tả middleware trong `internal/api/middleware` (architecture.md §II). Feature sẽ khiến docs này lệch thực tế → sẽ cập nhật `docs/system-design/architecture.md` trong Đợt 0/1 (xem tasks).

## Project Structure

### Documentation (this feature)

```text
specs/010-module-ownership-refactor/
├── plan.md              # File này
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output (không có data model — nêu rõ N/A)
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output (IAuthMiddleware, route-registry, tag mapping)
└── tasks.md             # Phase 2 output (/speckit-tasks - chưa tạo)
```

### Source Code (repository root)

```text
internal/
├── api/
│   ├── middleware/                    # [Đợt 0] XÓA hoàn toàn
│   └── routes/
│       ├── router.go                  # [giữ] NewEngine — chỉ gọi Init của 6 router module
│       ├── provider.go                # [giữ] AppRouter — 6 router module
│       └── <auth|me|admin|subscription|wardrobe|outfit|category|brand|fashion|dashboard>/   # [Đợt 1] XÓA
├── shared/
│   └── presentation/
│       ├── middleware/                # [Đợt 0] MỚI — cors, timeout, ratelimit, role, swagger_lock, error_handler
│       ├── response.go
│       └── sse.go
├── modules/
│   ├── identity/
│   │   └── presentation/
│   │       ├── middleware/auth.go     # [Đợt 0] MỚI (từ internal/api/middleware/auth.go)
│   │       ├── contract/auth_middleware.go  # [Đợt 0] MỚI — IAuthMiddleware
│   │       └── router/router.go       # [Đợt 1] MỚI — /auth, /me, /admin/users
│   ├── subscription/presentation/router/router.go        # [Đợt 1] /subscriptions
│   ├── wardrobe/presentation/router/router.go            # [Đợt 1] wardrobe-items, outfits, categories, system-catalog, admin
│   ├── dashboard/presentation/router/router.go           # [Đợt 1] analytics, me/brand/admin dashboard, drilldown
│   ├── brand/presentation/router/router.go               # [Đợt 1] brands, brand-portal, admin/brands
│   └── fashion/presentation/router/router.go             # [Đợt 1] /ai
├── tools/
│   └── swagger-split/main.go         # [Đợt 2] MỚI — Go script split spec
api/
└── swagger/
    ├── docs.go / swagger.json / swagger.yaml / index.html   # master spec (giữ)
    └── modules/<name>/swagger.json   # [Đợt 2] MỚI — 6 file per-module
```

**Structure Decision**: Theo kiến trúc modular monolith sẵn có (architecture.md §II/§IV). Router đặt trong `presentation/router/` của từng module theo canonical layout. Script split là Go tool (đặt `internal/tools/`), gọi qua Makefile — nhất quán cross-platform.

## Complexity Tracking

> Không cần — Constitution Check pass, không có violation cần justify.

---

# Phase 0 — Research

*Kết quả research chi tiết: [research.md](specs/010-module-ownership-refactor/research.md)*

Các quyết định nghiên cứu chính:

1. **Route-registry test (FR-017)**: Gin `engine.Routes()` trả `[]RoutesInfo` (Method + Path). Snapshot test so sánh tập route trước/sau refactor. Baseline chụp trước khi refactor, lưu dạng golden file. Test dựng `NewEngine` với handler nil-dep (constructor chỉ gán field, không deref lúc khởi tạo) → không cần DB.
2. **Swagger split script**: Chọn Go script (`go run ./internal/tools/swagger-split`), đọc master `swagger.json`, mapping tag→module (YAML/JSON), lọc `paths` → 6 file. Lý do: cross-platform, tái dùng JSON struct, `generate-check` diff được.
3. **Tag `Admin` dùng chung 3 module**: Q1 Option B yêu cầu mỗi tag thuộc đúng 1 module → tách `Admin` thành `Admin Identity` / `Admin Wardrobe` / `Admin Brand`.
4. **Import cycle**: `shared/presentation/middleware` KHÔNG import `internal/modules/*` (xác minh qua grep); `identity/presentation/contract` chỉ import `gin`. AuthMiddleware dùng thẳng `identityerrors` + `ITokenBlacklistService` — an toàn.
5. **Wire bind**: `wire.Bind(new(contract.IAuthMiddleware), new(*middleware.AuthMiddleware))` trong identity presentation provider.
6. **errorutils noise-candidates**: Chuỗi path `internal/api/middleware.CORSMiddleware` và `internal/api/routes.NewEngine.(*RateLimitMiddleware).Handle` phải cập nhật theo path mới; test `errorutils_test.go` đối chiếu stack frame cũng phải sửa.

# Phase 1 — Design & Contracts

## Data Model

**Không có thay đổi data model / database schema** (FR-016). Xem [data-model.md](specs/010-module-ownership-refactor/data-model.md).

## Contracts

- **[auth_middleware.md](specs/010-module-ownership-refactor/contracts/auth_middleware.md)** — interface `IAuthMiddleware` (Handle/OptionalHandle), nơi đặt, cách Wire bind, consumer nào dùng.
- **[route-registry.md](specs/010-module-ownership-refactor/contracts/route-registry.md)** — giao thức route-registry test: format golden file, cách sinh baseline, cách chạy.
- **[swagger-module-mapping.md](specs/010-module-ownership-refactor/contracts/swagger-module-mapping.md)** — mapping tag→module (sau realign), format spec per-module, cơ chế dropdown.

## Quickstart

- **[quickstart.md](specs/010-module-ownership-refactor/quickstart.md)** — hướng dẫn validate từng đợt: baseline route-registry, `make generate`, `make check`, kiểm tra dropdown swagger.
