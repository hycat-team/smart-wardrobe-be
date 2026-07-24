package admin

import (
	"smart-wardrobe-be/internal/api/middleware"
	dashboard_handler "smart-wardrobe-be/internal/modules/dashboard/presentation/handler"
	identity_handler "smart-wardrobe-be/internal/modules/identity/presentation/handler"
	wardrobe_handler "smart-wardrobe-be/internal/modules/wardrobe/presentation/handler"
	"smart-wardrobe-be/internal/shared/domain/constants/roleslug"
	shared_pres "smart-wardrobe-be/internal/shared/presentation"

	"github.com/gin-gonic/gin"
)

type AdminRouter struct {
	identityAdminHandler  *identity_handler.AdminHandler
	adminDashboardHandler *dashboard_handler.AdminDashboardHandler
	wardrobeItemHandler   *wardrobe_handler.WardrobeItemHandler
	categoryHandler       *wardrobe_handler.CategoryHandler
	authMiddleware        *middleware.AuthMiddleware
}

func NewRouter(
	identityAdminHandler *identity_handler.AdminHandler,
	adminDashboardHandler *dashboard_handler.AdminDashboardHandler,
	wardrobeItemHandler *wardrobe_handler.WardrobeItemHandler,
	categoryHandler *wardrobe_handler.CategoryHandler,
	authMiddleware *middleware.AuthMiddleware,
) *AdminRouter {
	return &AdminRouter{
		identityAdminHandler:  identityAdminHandler,
		adminDashboardHandler: adminDashboardHandler,
		wardrobeItemHandler:   wardrobeItemHandler,
		categoryHandler:       categoryHandler,
		authMiddleware:        authMiddleware,
	}
}

func (r *AdminRouter) Init(group *gin.RouterGroup) {
	admin := group.Group("/admin")
	admin.Use(r.authMiddleware.Handle(), middleware.RolesAuthorize(roleslug.Admin))

	adminDashboard := admin.Group("/dashboard")
	{
		adminDashboard.GET("/kpi-cards", shared_pres.WrapHandler(r.adminDashboardHandler.GetKPICards))
		adminDashboard.GET("/ai-margin", shared_pres.WrapHandler(r.adminDashboardHandler.GetAIMargin))
		adminDashboard.GET("/payos-reconciliation", shared_pres.WrapHandler(r.adminDashboardHandler.GetPayOSReconciliation))
		adminDashboard.GET("/overview", shared_pres.WrapHandler(r.adminDashboardHandler.GetOverview))

		// Expansion
		adminDashboard.GET("/subscriptions/distribution", shared_pres.WrapHandler(r.adminDashboardHandler.GetSubscriptionDistribution))
		adminDashboard.GET("/subscriptions/revenue-breakdown", shared_pres.WrapHandler(r.adminDashboardHandler.GetRevenueBreakdown))
		adminDashboard.GET("/ai/usage-by-operation", shared_pres.WrapHandler(r.adminDashboardHandler.GetAIUsageByOperation))
		adminDashboard.GET("/ai/error-breakdown", shared_pres.WrapHandler(r.adminDashboardHandler.GetAIErrorBreakdown))
		adminDashboard.GET("/subscriptions/renewal-stats", shared_pres.WrapHandler(r.adminDashboardHandler.GetRenewalStats))
	}

	adminUsers := admin.Group("/users")
	{
		adminUsers.GET("", shared_pres.WrapHandler(r.identityAdminHandler.GetUsers))
		adminUsers.PATCH("/:id/status", shared_pres.WrapHandler(r.identityAdminHandler.UpdateUserStatus))
	}

	adminWardrobe := admin.Group("/wardrobe-items")
	{
		adminWardrobe.GET("", shared_pres.WrapHandler(r.wardrobeItemHandler.GetCatalogItemsAdmin))
		adminWardrobe.PUT("/:id", shared_pres.WrapHandler(r.wardrobeItemHandler.UpdateCatalogItemAdmin))
		adminWardrobe.DELETE("/:id", shared_pres.WrapHandler(r.wardrobeItemHandler.DeleteCatalogItemAdmin))
		adminWardrobe.GET("/upload-signature", shared_pres.WrapHandler(r.wardrobeItemHandler.GetUploadSignature))
		adminWardrobe.POST("/batch-upload", shared_pres.WrapHandler(r.wardrobeItemHandler.BatchUploadWardrobeItems))
	}

	adminCategories := admin.Group("/categories")
	{
		adminCategories.GET("", shared_pres.WrapHandler(r.categoryHandler.GetCategoriesAdmin))
		adminCategories.GET("/:id", shared_pres.WrapHandler(r.categoryHandler.GetCategoryByIDAdmin))
		adminCategories.POST("", shared_pres.WrapHandler(r.categoryHandler.CreateCategoryAdmin))
		adminCategories.PUT("/:id", shared_pres.WrapHandler(r.categoryHandler.UpdateCategoryAdmin))
		adminCategories.DELETE("/:id", shared_pres.WrapHandler(r.categoryHandler.DeleteCategoryAdmin))
	}
}
