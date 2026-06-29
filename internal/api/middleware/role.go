package middleware

import (
	"slices"
	"smart-wardrobe-be/internal/shared/application/constants/apperror"
	"smart-wardrobe-be/internal/shared/domain/constants/identity/roleslug"
	"smart-wardrobe-be/pkg/utils/contextutils"

	"github.com/gin-gonic/gin"
)

func RolesAuthorize(allowedRoles ...roleslug.RoleSlug) gin.HandlerFunc {
	return func(c *gin.Context) {
		currentRole, err := contextutils.GetRoleSlug(c)
		if err != nil {
			c.Error(apperror.ErrInvalidAccessToken())
			c.Abort()
			return
		}

		allowed := slices.Contains(allowedRoles, currentRole)

		if !allowed {
			c.Error(apperror.ErrForbidden())
			c.Abort()
			return
		}

		c.Next()
	}
}
