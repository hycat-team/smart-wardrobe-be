# Contract: Brand Role Middleware (IBrandRoleMiddleware)

**Date**: 2026-08-17 | **Feature**: [spec.md](../spec.md)

Middleware kiểm tra brand membership (owner/staff) tập trung ở tầng HTTP, expose qua contract để module khác (dashboard) dùng mà không vi phạm module boundary.

---

## Interface (brand/presentation/contract/brand_role_middleware.go)

```go
package contract

import (
    "github.com/gin-gonic/gin"
    "smart-wardrobe-be/internal/modules/brand/domain/constants/brandmemberrole"
)

type BrandAuthorizeOption func(*brandAuthorizeOptions)

// WithAllowAdmin: cho phép role admin hệ thống bypass check membership (dùng cho brand dashboard).
func WithAllowAdmin() BrandAuthorizeOption

type IBrandRoleMiddleware interface {
    // PathAuthorize: lấy brandId từ path param (dùng cho /brand-portal/brands/:brandId/*).
    PathAuthorize(brandIDParam string, allowedRoles ...brandmemberrole.BrandMemberRole) gin.HandlerFunc
    // QueryAuthorize: lấy brandId từ query param (dùng cho /brand/dashboard/*).
    QueryAuthorize(brandIDParam string, allowedRoles ...brandmemberrole.BrandMemberRole) gin.HandlerFunc
}
```

## Implementation (brand/presentation/middleware/brand_role.go)

```go
type BrandRoleMiddleware struct {
    uc uc_interfaces.IBrandUseCase // GetUserBrandRole (cache-backed)
}

func NewBrandRoleMiddleware(uc uc_interfaces.IBrandUseCase) *BrandRoleMiddleware
```

Flow xử lý request:
1. `userID, _ := contextutils.GetUserId(c)`
2. Resolve `brandID` từ path/query param (`c.Param` / `c.Query`) → parse UUID; thiếu/sai → `apperror.NewBadRequest`.
3. Nếu `roleSlug == admin && allowAdmin` → set context `CtxBrandId`, `CtxBrandRole=admin` → `c.Next()`.
4. Ngược lại: `role, err := uc.GetUserBrandRole(ctx, userID, brandID)`:
   - `err != nil` (không phải member active / brand inactive) → `apperror.NewForbidden("Bạn không có quyền truy cập nhãn hàng này.")`.
   - `role` không nằm trong `allowedRoles` → `apperror.NewForbidden`.
5. Set context: `c.Set(contextutils.CtxBrandId, brandID)`, `c.Set(contextutils.CtxBrandRole, role)` → `c.Next()`.

## Context keys (internal/pkg/utils/contextutils)

- `CtxBrandId = "brandId"`
- `CtxBrandRole = "brandRole"`

Getter:
- `func GetBrandID(c *gin.Context) (uuid.UUID, error)`
- `func GetBrandRole(c *gin.Context) (brandmemberrole.BrandMemberRole, error)`

> Tuân thủ chuẩn context hiện có (hằng số key + getter typed) — không hardcode (A2/clarify).

## Cách dùng

**Brand router** (`brand/presentation/router/router.go`): chia 2 nhóm
- Nhóm `member` (Owner+Staff): toàn bộ route portal trừ 3 route không có `:brandId`.
- Nhóm `owner` (Owner): `PUT /loyalty/program`, `POST /loyalty/tiers`, `PUT /loyalty/tiers/:tierId`.

**Dashboard router** (`dashboard/presentation/router/router.go`): gắn `QueryAuthorize("brandId", Owner, Staff, WithAllowAdmin())` lên group `/brand/dashboard` và `/brand/dashboard/drilldown`.

## Wire

- Brand provider: `middleware.NewBrandRoleMiddleware` + `wire.Bind(contract.IBrandRoleMiddleware, *middleware.BrandRoleMiddleware)`.
- Dashboard provider/router nhận `contract.IBrandRoleMiddleware` (cross-module qua contract — G1).

## Kết quả

- Gỡ toàn bộ `RequireBrandRoleShared` trong usecase HTTP (~30 chỗ); giữ helper cho luồng nội bộ nếu cần.
- 403 trả đồng nhất; role đã xác minh có sẵn trong context (dùng cho column-level RBAC points-liability).