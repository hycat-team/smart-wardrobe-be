package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"smart-wardrobe-be/services/commerce/internal/modules/brand/application/dto"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/presentation/middleware"
	shared_dto "smart-wardrobe-be/shared/application/dto"
	"smart-wardrobe-be/shared/domain/constants/brandmemberrole"
	"smart-wardrobe-be/shared/domain/constants/roleslug"
	"smart-wardrobe-be/shared/pkg/utils/contextutils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockBrandUseCase struct {
	mock.Mock
}

func (m *mockBrandUseCase) GetUserBrandRole(ctx context.Context, userID, brandID uuid.UUID) (brandmemberrole.BrandMemberRole, error) {
	args := m.Called(ctx, userID, brandID)
	return args.Get(0).(brandmemberrole.BrandMemberRole), args.Error(1)
}

func (m *mockBrandUseCase) CreateBrandRequest(ctx context.Context, userID uuid.UUID, input dto.CreateBrandReq) (*dto.BrandRes, error) {
	return nil, nil
}
func (m *mockBrandUseCase) CreateBrandByAdmin(ctx context.Context, adminID uuid.UUID, input dto.CreateBrandReq) (*dto.BrandRes, error) {
	return nil, nil
}
func (m *mockBrandUseCase) UpdateBrandStatus(ctx context.Context, adminID uuid.UUID, brandID uuid.UUID, input dto.UpdateBrandStatusReq) (*dto.BrandRes, error) {
	return nil, nil
}
func (m *mockBrandUseCase) GetBrandsForAdmin(ctx context.Context, query dto.GetBrandsAdminQueryReq) (*dto.AdminBrandListRes, error) {
	return nil, nil
}
func (m *mockBrandUseCase) GetActiveBrands(ctx context.Context, query dto.GetActiveBrandsQueryReq) (*dto.PublicBrandBriefListRes, error) {
	return nil, nil
}
func (m *mockBrandUseCase) GetActiveBrand(ctx context.Context, brandID uuid.UUID) (*dto.BrandRes, error) {
	return nil, nil
}
func (m *mockBrandUseCase) GetBrandForPortal(ctx context.Context, userID uuid.UUID, brandID uuid.UUID) (*dto.PortalBrandRes, error) {
	return nil, nil
}
func (m *mockBrandUseCase) GetBrandsForPortalUser(ctx context.Context, userID uuid.UUID) ([]*dto.PortalBrandRes, error) {
	return nil, nil
}
func (m *mockBrandUseCase) GetBrandLogoUploadSignature(ctx context.Context, userID uuid.UUID) (*shared_dto.UploadSignatureResult, error) {
	return nil, nil
}
func (m *mockBrandUseCase) UpdateBrandImages(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, input dto.UpdateBrandImagesReq) (*dto.BrandRes, error) {
	return nil, nil
}
func (m *mockBrandUseCase) AddBrandMembers(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, input dto.AddBrandMembersReq) (*dto.AddBrandMembersRes, error) {
	return nil, nil
}
func (m *mockBrandUseCase) GetBrandMembers(ctx context.Context, userID uuid.UUID, brandID uuid.UUID) ([]*dto.BrandMemberRes, error) {
	return nil, nil
}
func (m *mockBrandUseCase) RequireBrandRole(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, allowedRoles ...brandmemberrole.BrandMemberRole) error {
	return nil
}
func (m *mockBrandUseCase) GetAdminBrandStats(ctx context.Context, snapshotDate time.Time) (*dto.BrandAdminStatsDTO, error) {
	return nil, nil
}
func (m *mockBrandUseCase) GetBrandAnalyticsStats(ctx context.Context, brandID uuid.UUID, snapshotDate time.Time) (*dto.BrandAnalyticsStatsDTO, error) {
	return nil, nil
}
func (m *mockBrandUseCase) InvalidateUserMembershipCache(ctx context.Context, userID uuid.UUID) error {
	return nil
}
func (m *mockBrandUseCase) InvalidateBrandMembersCache(ctx context.Context, brandID uuid.UUID) error {
	return nil
}
func (m *mockBrandUseCase) GetSampleLabFeedbacksDrillDown(ctx context.Context, itemID uuid.UUID, page, limit int) ([]dto.SampleFeedbackDrillDownDTO, int64, error) {
	return nil, 0, nil
}
func (m *mockBrandUseCase) GetItemFeedbackSummary(ctx context.Context, itemID uuid.UUID) (*dto.ItemFeedbackSummaryDTO, error) {
	return nil, nil
}
func (m *mockBrandUseCase) GetUserExpiringBenefitsCount(ctx context.Context, userID uuid.UUID) (int, error) {
	return 0, nil
}

func TestBrandRoleMiddleware_PathAuthorize(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("success when user has allowed role", func(t *testing.T) {
		mockUC := new(mockBrandUseCase)
		mw := middleware.NewBrandRoleMiddleware(mockUC)

		userID := uuid.New()
		brandID := uuid.New()

		mockUC.On("GetUserBrandRole", mock.Anything, userID, brandID).Return(brandmemberrole.Staff, nil)

		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set(contextutils.CtxUserId, userID.String())
			c.Next()
		})
		r.GET("/brands/:brandId/items", mw.PathAuthorize("brandId", brandmemberrole.Staff, brandmemberrole.Owner), func(c *gin.Context) {
			bID, _ := c.Get(contextutils.CtxBrandId)
			role, _ := c.Get(contextutils.CtxBrandRole)
			c.JSON(http.StatusOK, gin.H{"brandId": bID, "role": role})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/brands/"+brandID.String()+"/items", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockUC.AssertExpectations(t)
	})

	t.Run("forbidden when user does not have allowed role", func(t *testing.T) {
		mockUC := new(mockBrandUseCase)
		mw := middleware.NewBrandRoleMiddleware(mockUC)

		userID := uuid.New()
		brandID := uuid.New()

		mockUC.On("GetUserBrandRole", mock.Anything, userID, brandID).Return(brandmemberrole.Staff, nil)

		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Next()
			if len(c.Errors) > 0 {
				c.JSON(http.StatusForbidden, gin.H{"error": c.Errors.String()})
			}
		})
		r.Use(func(c *gin.Context) {
			c.Set(contextutils.CtxUserId, userID.String())
			c.Next()
		})
		r.GET("/brands/:brandId/settings", mw.PathAuthorize("brandId", brandmemberrole.Owner), func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/brands/"+brandID.String()+"/settings", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}

func TestBrandRoleMiddleware_QueryAuthorize_AdminBypass(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockUC := new(mockBrandUseCase)
	mw := middleware.NewBrandRoleMiddleware(mockUC)

	userID := uuid.New()
	brandID := uuid.New()

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(contextutils.CtxUserId, userID.String())
		c.Set(contextutils.CtxRoleSlug, string(roleslug.Admin))
		c.Next()
	})
	r.GET("/dashboard", mw.QueryAuthorize("brandId", brandmemberrole.Owner), func(c *gin.Context) {
		bID, _ := c.Get(contextutils.CtxBrandId)
		role, _ := c.Get(contextutils.CtxBrandRole)
		c.JSON(http.StatusOK, gin.H{"brandId": bID, "role": role})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/dashboard?brandId="+brandID.String(), nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	// UC should not be called due to admin bypass
	mockUC.AssertNotCalled(t, "GetUserBrandRole", mock.Anything, mock.Anything, mock.Anything)
}
