package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	branderrors "smart-wardrobe-be/services/commerce/internal/modules/brand/application/errors"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/presentation/middleware"
	"smart-wardrobe-be/shared/pkg/utils/contextutils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestBrandRefMiddleware_PathRewritesSlugToUUID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockUC := new(mockBrandUseCase)
	mw := middleware.NewBrandRefMiddleware(mockUC)

	brandID := uuid.New()
	mockUC.On("ResolveBrandRef", mock.Anything, "mori-studio").Return(brandID, nil)

	r := gin.New()
	r.GET("/brands/:brandId/items", mw.Path("brandId"), func(c *gin.Context) {
		ctxBrandID, _ := c.Get(contextutils.CtxBrandId)
		c.JSON(http.StatusOK, gin.H{
			"param": c.Param("brandId"),
			"ctx":   ctxBrandID,
		})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/brands/mori-studio/items", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), brandID.String())
	mockUC.AssertExpectations(t)
}

func TestBrandRefMiddleware_PathKeepsValidUUID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockUC := new(mockBrandUseCase)
	mw := middleware.NewBrandRefMiddleware(mockUC)

	brandID := uuid.New()
	mockUC.On("ResolveBrandRef", mock.Anything, brandID.String()).Return(brandID, nil)

	r := gin.New()
	r.GET("/brands/:brandId", mw.Path("brandId"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"param": c.Param("brandId")})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/brands/"+brandID.String(), nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), brandID.String())
}

func TestBrandRefMiddleware_PathAbortsOnUnknownSlug(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockUC := new(mockBrandUseCase)
	mw := middleware.NewBrandRefMiddleware(mockUC)

	mockUC.On("ResolveBrandRef", mock.Anything, "missing-brand").Return(uuid.Nil, branderrors.ErrBrandNotFound())

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Next()
		if len(c.Errors) > 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": c.Errors.String()})
		}
	})
	r.GET("/brands/:brandId", mw.Path("brandId"), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/brands/missing-brand", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestBrandRefMiddleware_PublicPathUsesActiveResolution(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockUC := new(mockBrandUseCase)
	mw := middleware.NewBrandRefMiddleware(mockUC)

	brandID := uuid.New()
	mockUC.On("ResolveActiveBrandRef", mock.Anything, "mori-studio").Return(brandID, nil)

	r := gin.New()
	r.GET("/brands/:brandId", mw.PublicPath("brandId"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"param": c.Param("brandId")})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/brands/mori-studio", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), brandID.String())
	mockUC.AssertNotCalled(t, "ResolveBrandRef", mock.Anything, mock.Anything)
}

func TestBrandRefMiddleware_QueryRewritesSlugToUUID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockUC := new(mockBrandUseCase)
	mw := middleware.NewBrandRefMiddleware(mockUC)

	brandID := uuid.New()
	mockUC.On("ResolveBrandRef", mock.Anything, "mori-studio").Return(brandID, nil)

	r := gin.New()
	r.GET("/brand/dashboard/kpi-cards", mw.Query("brandId"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"brandId": c.Query("brandId")})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/brand/dashboard/kpi-cards?brandId=mori-studio", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), brandID.String())
}

func TestBrandRefMiddleware_SkipsWhenParamMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockUC := new(mockBrandUseCase)
	mw := middleware.NewBrandRefMiddleware(mockUC)

	r := gin.New()
	r.GET("/me/brand-loyalties", mw.Path("brandId"), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/me/brand-loyalties", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	mockUC.AssertNotCalled(t, "ResolveBrandRef", mock.Anything, mock.Anything)
}
