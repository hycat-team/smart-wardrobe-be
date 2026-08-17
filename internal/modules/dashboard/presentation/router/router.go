package router

import (
	brand_contract "smart-wardrobe-be/internal/modules/brand/presentation/contract"
	dashboard_handler "smart-wardrobe-be/internal/modules/dashboard/presentation/handler"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/brandmemberrole"
	identity_contract "smart-wardrobe-be/internal/modules/identity/presentation/contract"
	"smart-wardrobe-be/internal/shared/domain/constants/roleslug"
	shared_pres "smart-wardrobe-be/internal/shared/presentation"
	shared_middleware "smart-wardrobe-be/internal/shared/presentation/middleware"

	"github.com/gin-gonic/gin"
)

type Router struct {
	analyticsHandler      *dashboard_handler.AnalyticsHandler
	adminDashboardHandler *dashboard_handler.AdminDashboardHandler
	brandDashboardHandler *dashboard_handler.BrandDashboardHandler
	userDashboardHandler  *dashboard_handler.UserDashboardHandler
	brandRole             brand_contract.IBrandRoleMiddleware
	auth                  identity_contract.IAuthMiddleware
}

func NewRouter(
	analyticsHandler *dashboard_handler.AnalyticsHandler,
	adminDashboardHandler *dashboard_handler.AdminDashboardHandler,
	brandDashboardHandler *dashboard_handler.BrandDashboardHandler,
	userDashboardHandler *dashboard_handler.UserDashboardHandler,
	brandRole brand_contract.IBrandRoleMiddleware,
	auth identity_contract.IAuthMiddleware,
) *Router {
	return &Router{
		analyticsHandler:      analyticsHandler,
		adminDashboardHandler: adminDashboardHandler,
		brandDashboardHandler: brandDashboardHandler,
		userDashboardHandler:  userDashboardHandler,
		brandRole:             brandRole,
		auth:                  auth,
	}
}

func (r *Router) Init(group *gin.RouterGroup) {
	analytics := group.Group("/dashboard/analytics")
	analytics.Use(r.auth.Handle())
	{
		analytics.POST("/query", shared_pres.WrapHandler(r.analyticsHandler.ExecuteDynamicQuery))
	}

	userDashboard := group.Group("/me/dashboard")
	userDashboard.Use(r.auth.Handle(), shared_middleware.RolesAuthorize(roleslug.User))
	{
		userDashboard.GET("/wardrobe-insights", shared_pres.WrapHandler(r.userDashboardHandler.GetWardrobeInsights))
		userDashboard.GET("/wardrobe/category-distribution", shared_pres.WrapHandler(r.userDashboardHandler.GetWardrobeCategoryDistribution))
	}

	brandDashboard := group.Group("/brand/dashboard")
	brandDashboard.Use(r.auth.Handle(), r.brandRole.QueryAuthorize("brandId", brandmemberrole.Owner, brandmemberrole.Staff))
	{
		brandDashboard.GET("/kpi-cards", shared_pres.WrapHandler(r.brandDashboardHandler.GetKPICards))
		brandDashboard.GET("/financials/points-liability", shared_pres.WrapHandler(r.brandDashboardHandler.GetPointsLiability))
		brandDashboard.GET("/operations/digital-sample-lab/:itemId", shared_pres.WrapHandler(r.brandDashboardHandler.GetDigitalSampleLabAnalytics))
		brandDashboard.GET("/customers/acquisition", shared_pres.WrapHandler(r.brandDashboardHandler.GetCustomerAcquisition))
		brandDashboard.GET("/loyalty/tier-distribution", shared_pres.WrapHandler(r.brandDashboardHandler.GetLoyaltyTierDistribution))
		brandDashboard.GET("/loyalty/point-expiry-forecast", shared_pres.WrapHandler(r.brandDashboardHandler.GetPointExpiryForecast))
		brandDashboard.GET("/benefits/redemption-analytics", shared_pres.WrapHandler(r.brandDashboardHandler.GetBenefitRedemptionAnalytics))
		brandDashboard.GET("/operations/catalog-stats", shared_pres.WrapHandler(r.brandDashboardHandler.GetCatalogStats))
		brandDashboard.GET("/customers/spend-segmentation", shared_pres.WrapHandler(r.brandDashboardHandler.GetCustomerSpendSegmentation))
	}

	adminDashboard := group.Group("/admin/dashboard")
	adminDashboard.Use(r.auth.Handle(), shared_middleware.RolesAuthorize(roleslug.Admin))
	{
		adminDashboard.GET("/kpi-cards", shared_pres.WrapHandler(r.adminDashboardHandler.GetKPICards))
		adminDashboard.GET("/ai-margin", shared_pres.WrapHandler(r.adminDashboardHandler.GetAIMargin))
		adminDashboard.GET("/payos-reconciliation", shared_pres.WrapHandler(r.adminDashboardHandler.GetPayOSReconciliation))
		adminDashboard.GET("/overview", shared_pres.WrapHandler(r.adminDashboardHandler.GetOverview))
		adminDashboard.GET("/subscriptions/distribution", shared_pres.WrapHandler(r.adminDashboardHandler.GetSubscriptionDistribution))
		adminDashboard.GET("/subscriptions/revenue-breakdown", shared_pres.WrapHandler(r.adminDashboardHandler.GetRevenueBreakdown))
		adminDashboard.GET("/ai/usage-by-operation", shared_pres.WrapHandler(r.adminDashboardHandler.GetAIUsageByOperation))
		adminDashboard.GET("/ai/error-breakdown", shared_pres.WrapHandler(r.adminDashboardHandler.GetAIErrorBreakdown))
		adminDashboard.GET("/subscriptions/renewal-stats", shared_pres.WrapHandler(r.adminDashboardHandler.GetRenewalStats))
	}

	adminDrill := group.Group("/admin/dashboard/drilldown")
	adminDrill.Use(r.auth.Handle(), shared_middleware.RolesAuthorize(roleslug.Admin))
	{
		adminDrill.GET("/ai-events", shared_pres.WrapHandler(r.analyticsHandler.GetAIEventsDrillDown))
	}

	brandDrill := group.Group("/brand/dashboard/drilldown")
	brandDrill.Use(r.auth.Handle(), r.brandRole.QueryAuthorize("brandId", brandmemberrole.Owner, brandmemberrole.Staff))
	{
		brandDrill.GET("/sample-lab-feedbacks", shared_pres.WrapHandler(r.analyticsHandler.GetSampleLabFeedbacksDrillDown))
	}
}
