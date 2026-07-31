# Contract: IAuthMiddleware (Đợt 0)

**Branch**: `010-module-ownership-refactor` | **Date**: 2026-08-01

## Mục đích

Expose khả năng xác thực JWT cho các module khác mà không khiến module import implementation của identity module. Đúng nguyên tắc "module chỉ phụ thuộc contract module khác" (Constitution II).

## Vị trí

- **Interface**: `internal/modules/identity/presentation/contract/auth_middleware.go` (package `contract`)
- **Implementation**: `internal/modules/identity/presentation/middleware/auth.go` (package `middleware`, di chuyển từ `internal/api/middleware/auth.go`)

## Interface

```go
package contract

import "github.com/gin-gonic/gin"

// IAuthMiddleware xác thực JWT cho request. Module khác (và NewEngine)
// chỉ phụ thuộc interface này, không import implementation.
type IAuthMiddleware interface {
    Handle() gin.HandlerFunc         // bắt buộc xác thực; 401 nếu thiếu/sai JWT
    OptionalHandle() gin.HandlerFunc // xác thực nếu có JWT; không fail nếu thiếu
}
```

## Wire Binding

Trong `internal/modules/identity/presentation/provider.go`:

```go
var ProviderSet = wire.NewSet(
    middleware.NewAuthMiddleware,
    wire.Bind(new(contract.IAuthMiddleware), new(*middleware.AuthMiddleware)),
    // ... handler hiện có
)
```

## Consumer

| Consumer | Cách dùng |
|---|---|
| Router các module (`wardrobe`, `brand`, `fashion`, `subscription`, `dashboard`) | `NewRouter(..., auth identity_contract.IAuthMiddleware, ...)` → `privateApi.Use(auth.Handle(), middleware.RolesAuthorize(...))` |
| `internal/api/routes/router.go` (NewEngine) | Không trực tiếp dùng AuthMiddleware; chỉ router module dùng |
| Auth router trong identity module | dùng implementation trực tiếp (cùng package) |

## Hành vi giữ nguyên (regression-safe)

- JWT hợp lệ → 200
- Thiếu JWT → 401 với message tiếng Việt
- Token blacklist → 401
- User blacklist → 401 "Tài khoản đã bị khoá hoặc vô hiệu hoá. Vui lòng liên hệ CSKH."
- `OptionalHandle` dùng cho route công khai có auth tùy chọn (vd `/brand-items/:itemId`)

## Ràng buộc

- Interface chỉ import `gin` — không import identity internal → không tạo cycle.
- Không thay đổi logic `authenticate()` trong `auth.go`.
