package contextutils

import (
	"smart-wardrobe-be/internal/modules/brand/domain/constants/brandmemberrole"
	"smart-wardrobe-be/internal/shared/application/constants/apperror"
	"smart-wardrobe-be/internal/shared/domain/constants/roleslug"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func GetUserId(c *gin.Context) (uuid.UUID, error) {
	val, exists := c.Get(CtxUserId)
	if !exists {
		return uuid.Nil, apperror.ErrUnauthorized()
	}

	uid, ok := val.(uuid.UUID)
	if !ok {
		strVal, ok := val.(string)
		if !ok {
			return uuid.Nil, apperror.ErrUnauthorized()
		}
		parsed, err := uuid.Parse(strVal)
		if err != nil {
			return uuid.Nil, apperror.ErrUnauthorized()
		}
		return parsed, nil
	}

	return uid, nil
}

func GetUserIdOptional(c *gin.Context) uuid.UUID {
	val, exists := c.Get(CtxUserId)
	if !exists {
		return uuid.Nil
	}
	switch v := val.(type) {
	case uuid.UUID:
		return v
	case string:
		parsed, err := uuid.Parse(v)
		if err != nil {
			return uuid.Nil
		}
		return parsed
	default:
		return uuid.Nil
	}
}

func GetRoleSlug(c *gin.Context) (roleslug.RoleSlug, error) {
	val, exists := c.Get(CtxRoleSlug)
	if !exists {
		return "", apperror.ErrUnauthorized()
	}

	role, ok := val.(string)
	if !ok {
		return "", apperror.ErrUnauthorized()
	}

	return roleslug.RoleSlug(role), nil
}

func GetEmail(c *gin.Context) (string, error) {
	val, exists := c.Get(CtxEmail)
	if !exists {
		return "", apperror.ErrUnauthorized()
	}

	email, ok := val.(string)
	if !ok {
		return "", apperror.ErrUnauthorized()
	}

	return email, nil
}

func GetAccessToken(c *gin.Context) (string, error) {
	val, exists := c.Get(CtxAccessToken)
	if !exists {
		return "", apperror.ErrUnauthorized()
	}

	token, ok := val.(string)
	if !ok {
		return "", apperror.ErrUnauthorized()
	}

	return token, nil
}

// GetBrandID returns the resolved brand ID stored on the context by the brand
// role middleware, or ErrUnauthorized if not present.
func GetBrandID(c *gin.Context) (uuid.UUID, error) {
	val, exists := c.Get(CtxBrandId)
	if !exists {
		return uuid.Nil, apperror.ErrUnauthorized()
	}
	switch v := val.(type) {
	case uuid.UUID:
		return v, nil
	case string:
		parsed, err := uuid.Parse(v)
		if err != nil {
			return uuid.Nil, apperror.ErrUnauthorized()
		}
		return parsed, nil
	default:
		return uuid.Nil, apperror.ErrUnauthorized()
	}
}

// GetBrandRole returns the verified brand member role stored on the context by
// the brand role middleware, or ErrUnauthorized if not present.
func GetBrandRole(c *gin.Context) (brandmemberrole.BrandMemberRole, error) {
	val, exists := c.Get(CtxBrandRole)
	if !exists {
		return "", apperror.ErrUnauthorized()
	}
	role, ok := val.(string)
	if !ok {
		return "", apperror.ErrUnauthorized()
	}
	return brandmemberrole.BrandMemberRole(role), nil
}
