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
		userBrands.GET("/:brandId/benefits", shared_pres.WrapHandler(r.brandHandler.ListActiveBenefitsForUser))
		userBrands.POST("/:brandId/benefits/:benefitId/redeem", shared_pres.WrapHandler(r.brandHandler.RedeemBenefit))
		userBrands.GET("/:brandId/conversation", shared_pres.WrapHandler(r.brandHandler.GetUserConversation))
		userBrands.POST("/:brandId/conversation/messages", shared_pres.WrapHandler(r.brandHandler.SendUserMessage))
		userBrands.GET("/:brandId/items", shared_pres.WrapHandler(r.brandHandler.ListBrandItemsForUser))
		userBrands.POST("/:brandId/items/:itemId/feedbacks", shared_pres.WrapHandler(r.brandHandler.SubmitSampleFeedback))
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
		portal.POST("/brands/:brandId/loyalty/points", shared_pres.WrapHandler(r.brandHandler.GrantLoyaltyPoints))
		portal.POST("/brands/:brandId/benefits", shared_pres.WrapHandler(r.brandHandler.CreateBrandBenefit))
		portal.GET("/brands/:brandId/benefits", shared_pres.WrapHandler(r.brandHandler.ListBrandBenefitsForStaff))
		portal.PATCH("/brands/:brandId/benefits/:benefitId/status", shared_pres.WrapHandler(r.brandHandler.UpdateBenefitStatus))
		portal.GET("/brands/:brandId/conversations", shared_pres.WrapHandler(r.brandHandler.ListBrandConversations))
		portal.GET("/brands/:brandId/conversations/:conversationId/messages", shared_pres.WrapHandler(r.brandHandler.ListConversationMessages))
		portal.POST("/brands/:brandId/conversations/:conversationId/messages", shared_pres.WrapHandler(r.brandHandler.SendStaffMessage))
		portal.POST("/brands/:brandId/items", shared_pres.WrapHandler(r.brandHandler.CreateBrandItem))
		portal.GET("/brands/:brandId/items", shared_pres.WrapHandler(r.brandHandler.GetBrandItemsForStaff))
		portal.PUT("/brands/:brandId/items/:itemId", shared_pres.WrapHandler(r.brandHandler.UpdateBrandItem))
		portal.GET("/brands/:brandId/items/:itemId/feedbacks", shared_pres.WrapHandler(r.brandHandler.GetBrandItemFeedbacks))
	}

	admin := group.Group("/admin/brands")
	admin.Use(r.authMiddleware.Handle(), middleware.RolesAuthorize(roleslug.Admin))
	{
		admin.POST("", shared_pres.WrapHandler(r.brandHandler.CreateBrandAdmin))
		admin.PATCH("/:brandId/status", shared_pres.WrapHandler(r.brandHandler.UpdateBrandStatusAdmin))
	}
}
