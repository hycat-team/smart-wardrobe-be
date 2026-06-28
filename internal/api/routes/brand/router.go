package brand

import (
	"smart-wardrobe-be/internal/api/middleware"
	brand_handler "smart-wardrobe-be/internal/modules/brand/presentation/handler"
	"smart-wardrobe-be/internal/shared/domain/constants/roleslug"
	shared_pres "smart-wardrobe-be/internal/shared/presentation"

	"github.com/gin-gonic/gin"
)

type BrandRouter struct {
	brandHandler   *brand_handler.BrandHandler
	authMiddleware *middleware.AuthMiddleware
}

func NewRouter(h *brand_handler.BrandHandler, m *middleware.AuthMiddleware) *BrandRouter {
	return &BrandRouter{brandHandler: h, authMiddleware: m}
}

func (r *BrandRouter) Init(group *gin.RouterGroup) {
	publicBrands := group.Group("/brands")
	{
		publicBrands.GET("", shared_pres.WrapHandler(r.brandHandler.GetActiveBrands))
	}

	userBrands := group.Group("/brands")
	userBrands.Use(r.authMiddleware.Handle(), middleware.RolesAuthorize(roleslug.User))
	{
		userBrands.POST("/:brandId/join-loyalty", shared_pres.WrapHandler(r.brandHandler.JoinLoyalty))
	}

	portal := group.Group("/brand-portal")
	portal.Use(r.authMiddleware.Handle(), middleware.RolesAuthorize(roleslug.User, roleslug.Admin))
	{
		portal.POST("/brands", shared_pres.WrapHandler(r.brandHandler.CreateBrandRequest))
		portal.GET("/brands/:brandId", shared_pres.WrapHandler(r.brandHandler.GetBrandForPortal))
		portal.POST("/brands/:brandId/members", shared_pres.WrapHandler(r.brandHandler.AddBrandMember))
		portal.GET("/brands/:brandId/members", shared_pres.WrapHandler(r.brandHandler.GetBrandMembers))
		portal.GET("/brands/:brandId/customers", shared_pres.WrapHandler(r.brandHandler.GetBrandCustomers))
		portal.POST("/brands/:brandId/customers/offline-purchase", shared_pres.WrapHandler(r.brandHandler.CreateOfflineCustomer))
	}

	admin := group.Group("/admin/brands")
	admin.Use(r.authMiddleware.Handle(), middleware.RolesAuthorize(roleslug.Admin))
	{
		admin.POST("", shared_pres.WrapHandler(r.brandHandler.CreateBrandAdmin))
		admin.PATCH("/:brandId/status", shared_pres.WrapHandler(r.brandHandler.UpdateBrandStatusAdmin))
	}
}
