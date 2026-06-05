package middleware

import (
	"smart-wardrobe-be/internal/shared/application/constants/apperror"
	"smart-wardrobe-be/internal/shared/domain/constants/roleslug"
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

		allowed := false
		for _, role := range allowedRoles {
			if currentRole == role {
				allowed = true
				break
			}
		}

		if !allowed {
			c.Error(apperror.ErrForbidden())
			c.Abort()
			return
		}

		c.Next()
	}
}
