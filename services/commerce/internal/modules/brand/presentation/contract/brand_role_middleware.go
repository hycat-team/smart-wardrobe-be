package contract

import (
	"smart-wardrobe-be/shared/domain/constants/brandmemberrole"

	"github.com/gin-gonic/gin"
)

// IBrandRoleMiddleware centralizes brand membership (owner/staff) authorization.
// Exposed as a contract so other modules (e.g. dashboard) can apply brand RBAC
// without depending on the middleware implementation, keeping module boundaries.
type IBrandRoleMiddleware interface {
	// PathAuthorize returns a handler that resolves brandId from a path param
	// (used by /brand-portal/brands/:brandId/*). No admin bypass: admins must
	// be real brand members to access the portal.
	PathAuthorize(brandIDParam string, allowedRoles ...brandmemberrole.BrandMemberRole) gin.HandlerFunc
	// QueryAuthorize returns a handler that resolves brandId from a query param
	// (used by /brand/dashboard/*). Allows system admins to bypass membership.
	QueryAuthorize(brandIDParam string, allowedRoles ...brandmemberrole.BrandMemberRole) gin.HandlerFunc
}
