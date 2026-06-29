package brand

import (
	"smart-wardrobe-be/internal/api/middleware"
	brand_handler "smart-wardrobe-be/internal/modules/brand/presentation/handler"
	"smart-wardrobe-be/internal/shared/domain/constants/identity/roleslug"
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
		publicBrands.GET("/:brandId", shared_pres.WrapHandler(r.brandHandler.GetActiveBrand))
	}

	userBrands := group.Group("/brands")
	userBrands.Use(r.authMiddleware.Handle(), middleware.RolesAuthorize(roleslug.User))
	{
		userBrands.POST("/claim", shared_pres.WrapHandler(r.brandHandler.ClaimOfflineAccount))
		userBrands.POST("/:brandId/join-loyalty", shared_pres.WrapHandler(r.brandHandler.JoinLoyalty))
		userBrands.GET("/:brandId/benefits", shared_pres.WrapHandler(r.brandHandler.ListActiveBenefitsForUser))
		userBrands.GET("/:brandId/conversation", shared_pres.WrapHandler(r.brandHandler.GetUserConversation))
		userBrands.POST("/:brandId/conversation/read", shared_pres.WrapHandler(r.brandHandler.MarkUserConversationRead))
		userBrands.POST("/:brandId/conversation/messages", shared_pres.WrapHandler(r.brandHandler.SendUserMessage))
		userBrands.GET("/:brandId/items", shared_pres.WrapHandler(r.brandHandler.ListBrandItemsForUser))
	}

	userBrandItems := group.Group("/brand-items")
	userBrandItems.Use(r.authMiddleware.Handle(), middleware.RolesAuthorize(roleslug.User))
	{
		userBrandItems.GET("/:itemId", shared_pres.WrapHandler(r.brandHandler.GetBrandItemForUser))
		userBrandItems.POST("/:itemId/feedbacks", shared_pres.WrapHandler(r.brandHandler.SubmitSampleFeedback))
	}

	userBrandBenefits := group.Group("/brand-benefits")
	userBrandBenefits.Use(r.authMiddleware.Handle(), middleware.RolesAuthorize(roleslug.User))
	{
		userBrandBenefits.GET("/:benefitId", shared_pres.WrapHandler(r.brandHandler.GetActiveBenefitForUser))
		userBrandBenefits.POST("/:benefitId/redeem", shared_pres.WrapHandler(r.brandHandler.RedeemBenefit))
	}

	meBrands := group.Group("/me")
	meBrands.Use(r.authMiddleware.Handle(), middleware.RolesAuthorize(roleslug.User))
	{
		meBrands.GET("/brand-loyalties", shared_pres.WrapHandler(r.brandHandler.ListUserBrandLoyalties))
		meBrands.GET("/brand-loyalties/:brandId", shared_pres.WrapHandler(r.brandHandler.GetUserBrandLoyalty))
		meBrands.GET("/brand-loyalties/:brandId/transactions", shared_pres.WrapHandler(r.brandHandler.GetUserBrandLoyaltyTransactions))
		meBrands.GET("/brand-loyalties/:brandId/lots", shared_pres.WrapHandler(r.brandHandler.GetUserBrandLoyaltyLots))
		meBrands.GET("/benefit-redemptions", shared_pres.WrapHandler(r.brandHandler.ListBenefitRedemptionsForUser))
	}

	portal := group.Group("/brand-portal")
	portal.Use(r.authMiddleware.Handle(), middleware.RolesAuthorize(roleslug.User, roleslug.Admin))
	{
		portal.GET("/me/brands", shared_pres.WrapHandler(r.brandHandler.GetMyPortalBrands))
		portal.POST("/brands", shared_pres.WrapHandler(r.brandHandler.CreateBrandRequest))
		portal.GET("/brands/logo-upload-signature", shared_pres.WrapHandler(r.brandHandler.GetBrandLogoUploadSignature))
		portal.GET("/brands/:brandId", shared_pres.WrapHandler(r.brandHandler.GetBrandForPortal))
		portal.PATCH("/brands/:brandId/logo", shared_pres.WrapHandler(r.brandHandler.UpdateBrandLogo))
		portal.POST("/brands/:brandId/members", shared_pres.WrapHandler(r.brandHandler.AddBrandMember))
		portal.GET("/brands/:brandId/members", shared_pres.WrapHandler(r.brandHandler.GetBrandMembers))
		portal.GET("/brands/:brandId/customers", shared_pres.WrapHandler(r.brandHandler.GetBrandCustomers))
		portal.GET("/brands/:brandId/customers/:customerId", shared_pres.WrapHandler(r.brandHandler.GetBrandCustomer))
		portal.POST("/brands/:brandId/customers/:customerId/claim-token", shared_pres.WrapHandler(r.brandHandler.CreateClaimToken))
		portal.GET("/brands/:brandId/customers/:customerId/claim-tokens", shared_pres.WrapHandler(r.brandHandler.ListClaimTokens))
		portal.POST("/brands/:brandId/customers/:customerId/claim-tokens/:claimId/revoke", shared_pres.WrapHandler(r.brandHandler.RevokeClaimToken))
		portal.POST("/brands/:brandId/customers/offline-purchase", shared_pres.WrapHandler(r.brandHandler.CreateOfflineCustomer))
		portal.POST("/brands/:brandId/loyalty/points", shared_pres.WrapHandler(r.brandHandler.GrantLoyaltyPoints))
		portal.GET("/brands/:brandId/loyalty/accounts/:accountId/transactions", shared_pres.WrapHandler(r.brandHandler.GetLoyaltyAccountTransactionsForStaff))
		portal.GET("/brands/:brandId/loyalty/accounts/:accountId/lots", shared_pres.WrapHandler(r.brandHandler.GetLoyaltyAccountLotsForStaff))
		portal.GET("/brands/:brandId/loyalty/program", shared_pres.WrapHandler(r.brandHandler.GetLoyaltyProgramForStaff))
		portal.GET("/brands/:brandId/loyalty/tiers", shared_pres.WrapHandler(r.brandHandler.GetLoyaltyTiersForStaff))
		portal.POST("/brands/:brandId/benefits", shared_pres.WrapHandler(r.brandHandler.CreateBrandBenefit))
		portal.GET("/brands/:brandId/benefits", shared_pres.WrapHandler(r.brandHandler.ListBrandBenefitsForStaff))
		portal.PATCH("/brands/:brandId/benefits/:benefitId/status", shared_pres.WrapHandler(r.brandHandler.UpdateBenefitStatus))
		portal.GET("/brands/:brandId/conversations", shared_pres.WrapHandler(r.brandHandler.ListBrandConversations))
		portal.GET("/brands/:brandId/conversations/:conversationId/messages", shared_pres.WrapHandler(r.brandHandler.ListConversationMessages))
		portal.POST("/brands/:brandId/conversations/:conversationId/read", shared_pres.WrapHandler(r.brandHandler.MarkStaffConversationRead))
		portal.POST("/brands/:brandId/conversations/:conversationId/close", shared_pres.WrapHandler(r.brandHandler.CloseConversation))
		portal.POST("/brands/:brandId/conversations/:conversationId/reopen", shared_pres.WrapHandler(r.brandHandler.ReopenConversation))
		portal.POST("/brands/:brandId/conversations/:conversationId/messages", shared_pres.WrapHandler(r.brandHandler.SendStaffMessage))
		portal.POST("/brands/:brandId/items", shared_pres.WrapHandler(r.brandHandler.CreateBrandItem))
		portal.GET("/brands/:brandId/items", shared_pres.WrapHandler(r.brandHandler.GetBrandItemsForStaff))
		portal.GET("/brands/:brandId/items/upload-signature", shared_pres.WrapHandler(r.brandHandler.GetBrandItemUploadSignature))
		portal.GET("/brands/:brandId/items/:itemId", shared_pres.WrapHandler(r.brandHandler.GetBrandItemForStaff))
		portal.PUT("/brands/:brandId/items/:itemId", shared_pres.WrapHandler(r.brandHandler.UpdateBrandItem))
		portal.PATCH("/brands/:brandId/items/:itemId/status", shared_pres.WrapHandler(r.brandHandler.UpdateBrandItemStatus))
		portal.GET("/brands/:brandId/items/:itemId/feedbacks", shared_pres.WrapHandler(r.brandHandler.GetBrandItemFeedbacks))
	}

	admin := group.Group("/admin/brands")
	admin.Use(r.authMiddleware.Handle(), middleware.RolesAuthorize(roleslug.Admin))
	{
		admin.GET("", shared_pres.WrapHandler(r.brandHandler.GetBrandsAdmin))
		admin.POST("", shared_pres.WrapHandler(r.brandHandler.CreateBrandAdmin))
		admin.PATCH("/:brandId/status", shared_pres.WrapHandler(r.brandHandler.UpdateBrandStatusAdmin))
	}
}
