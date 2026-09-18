package router

import (
	identity_contract "smart-wardrobe-be/services/account/contract/identity"
	brand_handler "smart-wardrobe-be/services/commerce/internal/modules/brand/presentation/handler"
	"smart-wardrobe-be/shared/domain/constants/roleslug"
	shared_pres "smart-wardrobe-be/shared/presentation"
	shared_middleware "smart-wardrobe-be/shared/presentation/middleware"

	brand_contract "smart-wardrobe-be/services/commerce/internal/modules/brand/presentation/contract"
	"smart-wardrobe-be/shared/domain/constants/brandmemberrole"

	"github.com/gin-gonic/gin"
)

type Router struct {
	brandHandler      *brand_handler.BrandHandler
	adminBrandHandler *brand_handler.AdminBrandHandler
	brandRole         brand_contract.IBrandRoleMiddleware
	auth              identity_contract.IAuthMiddleware
}

func NewRouter(
	brandHandler *brand_handler.BrandHandler,
	adminBrandHandler *brand_handler.AdminBrandHandler,
	brandRole brand_contract.IBrandRoleMiddleware,
	auth identity_contract.IAuthMiddleware,
) *Router {
	return &Router{
		brandHandler:      brandHandler,
		adminBrandHandler: adminBrandHandler,
		brandRole:         brandRole,
		auth:              auth,
	}
}

func (r *Router) Init(group *gin.RouterGroup) {
	publicBrands := group.Group("/brands")
	{
		publicBrands.GET("", shared_pres.WrapHandler(r.brandHandler.GetActiveBrands))
		publicBrands.GET("/:brandId", shared_pres.WrapHandler(r.brandHandler.GetActiveBrand))
		publicBrands.GET("/:brandId/items", shared_pres.WrapHandler(r.brandHandler.ListBrandProducts))
		publicBrands.GET("/:brandId/benefits", shared_pres.WrapHandler(r.brandHandler.ListActiveBenefitsForUser))
		publicBrands.GET("/:brandId/loyalty/tiers/:tierId", shared_pres.WrapHandler(r.brandHandler.GetLoyaltyTierDetailForUser))
	}

	userBrands := group.Group("/brands")
	userBrands.Use(r.auth.Handle(), shared_middleware.RolesAuthorize(roleslug.User))
	{
		userBrands.POST("/claim", shared_pres.WrapHandler(r.brandHandler.ClaimOfflineAccount))
		userBrands.POST("/:brandId/join-loyalty", shared_pres.WrapHandler(r.brandHandler.JoinLoyalty))
		userBrands.GET("/:brandId/conversation", shared_pres.WrapHandler(r.brandHandler.GetUserConversation))
		userBrands.POST("/:brandId/conversation/read", shared_pres.WrapHandler(r.brandHandler.MarkUserConversationRead))
		userBrands.POST("/:brandId/conversation/messages", shared_pres.WrapHandler(r.brandHandler.SendUserMessage))
		userBrands.GET("/:brandId/items/samples", shared_pres.WrapHandler(r.brandHandler.ListBrandSamples))
	}

	userBrandItems := group.Group("/brand-items")
	userBrandItems.Use(r.auth.Handle(), shared_middleware.RolesAuthorize(roleslug.User))
	{
		userBrandItems.POST("/:itemId/feedbacks", shared_pres.WrapHandler(r.brandHandler.SubmitSampleFeedback))
	}

	optAuthBrandItems := group.Group("/brand-items")
	optAuthBrandItems.Use(r.auth.OptionalHandle())
	{
		optAuthBrandItems.GET("/:itemId", shared_pres.WrapHandler(r.brandHandler.GetBrandItemForUser))
	}

	userBrandBenefits := group.Group("/brand-benefits")
	userBrandBenefits.Use(r.auth.Handle(), shared_middleware.RolesAuthorize(roleslug.User))
	{
		userBrandBenefits.GET("/:benefitId", shared_pres.WrapHandler(r.brandHandler.GetActiveBenefitForUser))
		userBrandBenefits.POST("/:benefitId/redeem", shared_pres.WrapHandler(r.brandHandler.RedeemBenefit))
	}

	meBrands := group.Group("/me")
	meBrands.Use(r.auth.Handle(), shared_middleware.RolesAuthorize(roleslug.User))
	{
		meBrands.GET("/brand-loyalties", shared_pres.WrapHandler(r.brandHandler.ListUserBrandLoyalties))
		meBrands.GET("/brand-loyalties/:brandId", shared_pres.WrapHandler(r.brandHandler.GetUserBrandLoyalty))
		meBrands.GET("/brand-loyalties/:brandId/transactions", shared_pres.WrapHandler(r.brandHandler.GetUserBrandLoyaltyTransactions))
		meBrands.GET("/brand-loyalties/:brandId/lots", shared_pres.WrapHandler(r.brandHandler.GetUserBrandLoyaltyLots))
		meBrands.GET("/benefit-redemptions", shared_pres.WrapHandler(r.brandHandler.ListBenefitRedemptionsForUser))
	}

	portal := group.Group("/brand-portal")
	portal.Use(r.auth.Handle(), shared_middleware.RolesAuthorize(roleslug.User, roleslug.Admin))
	{
		portal.GET("/me/brands", shared_pres.WrapHandler(r.brandHandler.GetMyPortalBrands))
		portal.POST("/brands", shared_pres.WrapHandler(r.brandHandler.CreateBrandRequest))
		portal.GET("/brands/profile-images/upload-signature", shared_pres.WrapHandler(r.brandHandler.GetBrandLogoUploadSignature))
	}

	// Brand RBAC is centralized here via BrandRoleMiddleware: routes requiring a
	// real brand membership (owner/staff) go under memberGroup; owner-only
	// configuration routes (loyalty program/tiers) go under ownerGroup.
	portalMember := group.Group("/brand-portal")
	portalMember.Use(r.auth.Handle(), shared_middleware.RolesAuthorize(roleslug.User, roleslug.Admin), r.brandRole.PathAuthorize("brandId", brandmemberrole.Owner, brandmemberrole.Staff))
	{
		portalMember.GET("/brands/:brandId", shared_pres.WrapHandler(r.brandHandler.GetBrandForPortal))
		portalMember.PATCH("/brands/:brandId/profile-images", shared_pres.WrapHandler(r.brandHandler.UpdateBrandImages))
		portalMember.POST("/brands/:brandId/members", shared_pres.WrapHandler(r.brandHandler.AddBrandMember))
		portalMember.GET("/brands/:brandId/members", shared_pres.WrapHandler(r.brandHandler.GetBrandMembers))
		portalMember.GET("/brands/:brandId/customers", shared_pres.WrapHandler(r.brandHandler.GetBrandCustomers))
		portalMember.GET("/brands/:brandId/customers/:customerId", shared_pres.WrapHandler(r.brandHandler.GetBrandCustomer))
		portalMember.POST("/brands/:brandId/customers/:customerId/claim-token", shared_pres.WrapHandler(r.brandHandler.CreateClaimToken))
		portalMember.GET("/brands/:brandId/customers/:customerId/claim-tokens", shared_pres.WrapHandler(r.brandHandler.ListClaimTokens))
		portalMember.POST("/brands/:brandId/customers/:customerId/claim-tokens/:claimId/revoke", shared_pres.WrapHandler(r.brandHandler.RevokeClaimToken))
		portalMember.POST("/brands/:brandId/customers/offline-purchase", shared_pres.WrapHandler(r.brandHandler.CreateOfflineCustomer))
		portalMember.POST("/brands/:brandId/loyalty/points", shared_pres.WrapHandler(r.brandHandler.GrantLoyaltyPoints))
		portalMember.GET("/brands/:brandId/loyalty/accounts/:accountId/transactions", shared_pres.WrapHandler(r.brandHandler.GetLoyaltyAccountTransactionsForStaff))
		portalMember.GET("/brands/:brandId/loyalty/accounts/:accountId/lots", shared_pres.WrapHandler(r.brandHandler.GetLoyaltyAccountLotsForStaff))
		portalMember.GET("/brands/:brandId/loyalty/program", shared_pres.WrapHandler(r.brandHandler.GetLoyaltyProgramForStaff))
		portalMember.GET("/brands/:brandId/loyalty/tiers", shared_pres.WrapHandler(r.brandHandler.GetLoyaltyTiersForStaff))
		portalMember.GET("/brands/:brandId/loyalty/tiers/:tierId", shared_pres.WrapHandler(r.brandHandler.GetLoyaltyTierDetailForStaff))
		portalMember.POST("/brands/:brandId/benefits", shared_pres.WrapHandler(r.brandHandler.CreateBrandBenefit))
		portalMember.GET("/brands/:brandId/benefits", shared_pres.WrapHandler(r.brandHandler.ListBrandBenefitsForStaff))
		portalMember.PATCH("/brands/:brandId/benefits/:benefitId/status", shared_pres.WrapHandler(r.brandHandler.UpdateBenefitStatus))
		portalMember.GET("/brands/:brandId/conversations", shared_pres.WrapHandler(r.brandHandler.ListBrandConversations))
		portalMember.GET("/brands/:brandId/conversations/:conversationId/messages", shared_pres.WrapHandler(r.brandHandler.ListConversationMessages))
		portalMember.POST("/brands/:brandId/conversations/:conversationId/read", shared_pres.WrapHandler(r.brandHandler.MarkStaffConversationRead))
		portalMember.POST("/brands/:brandId/conversations/:conversationId/close", shared_pres.WrapHandler(r.brandHandler.CloseConversation))
		portalMember.POST("/brands/:brandId/conversations/:conversationId/reopen", shared_pres.WrapHandler(r.brandHandler.ReopenConversation))
		portalMember.POST("/brands/:brandId/conversations/:conversationId/messages", shared_pres.WrapHandler(r.brandHandler.SendStaffMessage))
		portalMember.POST("/brands/:brandId/items", shared_pres.WrapHandler(r.brandHandler.CreateBrandItem))
		portalMember.GET("/brands/:brandId/items/tasks/:taskId/sse", shared_pres.WrapHandler(r.brandHandler.SubscribeBrandItemTaskSSE))
		portalMember.GET("/brands/:brandId/items", shared_pres.WrapHandler(r.brandHandler.GetBrandItemsForStaff))
		portalMember.GET("/brands/:brandId/items/upload-signature", shared_pres.WrapHandler(r.brandHandler.GetBrandItemUploadSignature))
		portalMember.GET("/brands/:brandId/items/:itemId", shared_pres.WrapHandler(r.brandHandler.GetBrandItemForStaff))
		portalMember.PUT("/brands/:brandId/items/:itemId", shared_pres.WrapHandler(r.brandHandler.UpdateBrandItem))
		portalMember.PATCH("/brands/:brandId/items/:itemId/status", shared_pres.WrapHandler(r.brandHandler.UpdateBrandItemStatus))
		portalMember.GET("/brands/:brandId/items/:itemId/feedbacks", shared_pres.WrapHandler(r.brandHandler.GetBrandItemFeedbacks))
	}

	portalOwner := group.Group("/brand-portal")
	portalOwner.Use(r.auth.Handle(), shared_middleware.RolesAuthorize(roleslug.User, roleslug.Admin), r.brandRole.PathAuthorize("brandId", brandmemberrole.Owner))
	{
		portalOwner.PUT("/brands/:brandId/loyalty/program", shared_pres.WrapHandler(r.brandHandler.UpsertLoyaltyProgram))
		portalOwner.POST("/brands/:brandId/loyalty/tiers", shared_pres.WrapHandler(r.brandHandler.CreateLoyaltyTier))
		portalOwner.PUT("/brands/:brandId/loyalty/tiers/:tierId", shared_pres.WrapHandler(r.brandHandler.UpdateLoyaltyTier))
	}

	admin := group.Group("/admin/brands")
	admin.Use(r.auth.Handle(), shared_middleware.RolesAuthorize(roleslug.Admin))
	{
		admin.GET("", shared_pres.WrapHandler(r.brandHandler.GetBrandsAdmin))
		admin.POST("", shared_pres.WrapHandler(r.brandHandler.CreateBrandAdmin))
		admin.PATCH("/:brandId/status", shared_pres.WrapHandler(r.brandHandler.UpdateBrandStatusAdmin))

		admin.GET("/:brandId/members", shared_pres.WrapHandler(r.adminBrandHandler.ListMembers))
		admin.POST("/:brandId/members", shared_pres.WrapHandler(r.adminBrandHandler.AddStaff))
		admin.PATCH("/:brandId/members/:memberId", shared_pres.WrapHandler(r.adminBrandHandler.UpdateMember))
		admin.DELETE("/:brandId/members/:memberId", shared_pres.WrapHandler(r.adminBrandHandler.RemoveMember))
		admin.POST("/:brandId/transfer-ownership", shared_pres.WrapHandler(r.adminBrandHandler.TransferOwnership))
	}
}
