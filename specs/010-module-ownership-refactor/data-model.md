# Data Model — Củng cố Ranh Giới Module

**Branch**: `010-module-ownership-refactor` | **Date**: 2026-08-01

## Kết luận

**Không có thay đổi data model / database schema trong feature này** (theo FR-016 và Assumptions trong spec).

- Không tạo bảng mới, không đổi cột, không đổi migration.
- Không tách database schema theo module — DB vẫn dùng chung một database/schema như hiện tại.
- Entity ownership theo module (đã có trong `internal/modules/<name>/domain/entities/`) không thay đổi.

## Lý do

Đây là refactor thuần **tầng code** (presentation layer): di chuyển middleware, đưa endpoint về module, tách swagger. Không đụng tới tầng domain/infrastructure/persistence.

## Các "thực thể" liên quan (phi-data-model)

| Khái niệm | Vị trí mới | Ghi chú |
|---|---|---|
| Shared middleware | `internal/shared/presentation/middleware/` | cors, timeout, ratelimit, role, swagger_lock, error_handler |
| Auth middleware | `internal/modules/identity/presentation/middleware/auth.go` | thuộc identity |
| `IAuthMiddleware` interface | `internal/modules/identity/presentation/contract/auth_middleware.go` | contract, chỉ import gin |
| Module router | `internal/modules/<name>/presentation/router/router.go` | 6 module |
| `AppRouter` | `internal/api/routes/provider.go` | chỉ còn 6 router module |
| Spec swagger per-module | `api/swagger/modules/<name>/swagger.json` | 6 file, sinh từ master |

## Validation

Không có validation rule data nào mới. RBAC (roleslug) và auth flow giữ nguyên.
