package contextutils

import (
	"smart-wardrobe-be/internal/shared/application/constants/apperror"
	"smart-wardrobe-be/internal/shared/domain/constants/identity/roleslug"

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
