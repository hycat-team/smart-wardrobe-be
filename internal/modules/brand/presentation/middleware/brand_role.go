package middleware

import (
	"slices"

	branderrors "smart-wardrobe-be/internal/modules/brand/application/errors"
	uc_interfaces "smart-wardrobe-be/internal/modules/brand/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/brandmemberrole"
	"smart-wardrobe-be/internal/pkg/utils/contextutils"
	"smart-wardrobe-be/internal/shared/application/constants/apperror"
	"smart-wardrobe-be/internal/shared/domain/constants/roleslug"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// BrandRoleMiddleware centralizes the brand membership (owner/staff) checks
// for brand portal and brand dashboard routes.
type BrandRoleMiddleware struct {
	brandUC uc_interfaces.IBrandUseCase
}

func NewBrandRoleMiddleware(brandUC uc_interfaces.IBrandUseCase) *BrandRoleMiddleware {
	return &BrandRoleMiddleware{brandUC: brandUC}
}

// PathAuthorize builds a handler that resolves brandId from a path parameter.
// No admin bypass: admins must be real brand members to access the portal.
func (m *BrandRoleMiddleware) PathAuthorize(brandIDParam string, allowedRoles ...brandmemberrole.BrandMemberRole) gin.HandlerFunc {
	return m.authorize(true, brandIDParam, false, allowedRoles...)
}

// QueryAuthorize builds a handler that resolves brandId from a query parameter.
// System admins bypass membership (used for brand dashboard endpoints).
func (m *BrandRoleMiddleware) QueryAuthorize(brandIDParam string, allowedRoles ...brandmemberrole.BrandMemberRole) gin.HandlerFunc {
	return m.authorize(false, brandIDParam, true, allowedRoles...)
}

func (m *BrandRoleMiddleware) authorize(fromPath bool, brandIDParam string, allowAdmin bool, allowedRoles ...brandmemberrole.BrandMemberRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		var raw string
		if fromPath {
			raw = c.Param(brandIDParam)
		} else {
			raw = c.Query(brandIDParam)
		}
		if raw == "" {
			c.Error(apperror.NewBadRequest("Thiếu brandId để xác thực quyền truy cập."))
			c.Abort()
			return
		}
		brandID, err := uuid.Parse(raw)
		if err != nil {
			c.Error(apperror.NewBadRequest("brandId không hợp lệ."))
			c.Abort()
			return
		}

		userID, err := contextutils.GetUserId(c)
		if err != nil {
			c.Error(err)
			c.Abort()
			return
		}

		roleSlug, _ := contextutils.GetRoleSlug(c)
		if allowAdmin && roleSlug == roleslug.Admin {
			c.Set(contextutils.CtxBrandId, brandID)
			c.Set(contextutils.CtxBrandRole, brandmemberrole.BrandMemberRole("admin"))
			c.Next()
			return
		}

		role, err := m.brandUC.GetUserBrandRole(c.Request.Context(), userID, brandID)
		if err != nil {
			c.Error(err)
			c.Abort()
			return
		}
		if !slices.Contains(allowedRoles, role) {
			c.Error(branderrors.ErrBrandPortalForbidden())
			c.Abort()
			return
		}

		c.Set(contextutils.CtxBrandId, brandID)
		c.Set(contextutils.CtxBrandRole, role)
		c.Next()
	}
}
