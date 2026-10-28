package middleware

import (
	uc_interfaces "smart-wardrobe-be/services/commerce/internal/modules/brand/application/interface/usecase"
	"smart-wardrobe-be/shared/pkg/utils/contextutils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type BrandRefMiddleware struct {
	brandUC uc_interfaces.IBrandUseCase
}

func NewBrandRefMiddleware(brandUC uc_interfaces.IBrandUseCase) *BrandRefMiddleware {
	return &BrandRefMiddleware{brandUC: brandUC}
}

func (m *BrandRefMiddleware) Path(brandIDParam string) gin.HandlerFunc {
	return m.resolve(brandIDParam, false, false)
}

func (m *BrandRefMiddleware) PublicPath(brandIDParam string) gin.HandlerFunc {
	return m.resolve(brandIDParam, false, true)
}

func (m *BrandRefMiddleware) Query(brandIDParam string) gin.HandlerFunc {
	return m.resolve(brandIDParam, true, false)
}

func (m *BrandRefMiddleware) resolve(brandIDParam string, fromQuery bool, activeOnly bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := c.Param(brandIDParam)
		if fromQuery {
			raw = c.Request.URL.Query().Get(brandIDParam)
		}
		if raw == "" {
			c.Next()
			return
		}

		brandID, err := m.resolveRef(c, raw, activeOnly)
		if err != nil {
			c.Error(err)
			c.Abort()
			return
		}

		if fromQuery {
			query := c.Request.URL.Query()
			query.Set(brandIDParam, brandID.String())
			c.Request.URL.RawQuery = query.Encode()
			c.Query(brandIDParam)
		} else {
			for i := range c.Params {
				if c.Params[i].Key == brandIDParam {
					c.Params[i].Value = brandID.String()
					break
				}
			}
		}

		c.Set(contextutils.CtxBrandId, brandID)
		c.Next()
	}
}

func (m *BrandRefMiddleware) resolveRef(c *gin.Context, raw string, activeOnly bool) (uuid.UUID, error) {
	if activeOnly {
		return m.brandUC.ResolveActiveBrandRef(c.Request.Context(), raw)
	}
	return m.brandUC.ResolveBrandRef(c.Request.Context(), raw)
}
