package dashboard

import (
	"smart-wardrobe-be/internal/api/middleware"
	dashboard_handler "smart-wardrobe-be/internal/modules/dashboard/presentation/handler"
	"smart-wardrobe-be/internal/shared/domain/constants/roleslug"
	shared_pres "smart-wardrobe-be/internal/shared/presentation"

	"github.com/gin-gonic/gin"
)

type DashboardRouter struct {
	analyticsHandler      *dashboard_handler.AnalyticsHandler
	adminDashboardHandler *dashboard_handler.AdminDashboardHandler
	brandDashboardHandler *dashboard_handler.BrandDashboardHandler
	authMiddleware        *middleware.AuthMiddleware
}

func NewRouter(
	analyticsHandler *dashboard_handler.AnalyticsHandler,
	adminDashboardHandler *dashboard_handler.AdminDashboardHandler,
	brandDashboardHandler *dashboard_handler.BrandDashboardHandler,
	authMiddleware *middleware.AuthMiddleware,
) *DashboardRouter {
	return &DashboardRouter{
		analyticsHandler:      analyticsHandler,
		adminDashboardHandler: adminDashboardHandler,
		brandDashboardHandler: brandDashboardHandler,
		authMiddleware:        authMiddleware,
	}
}

func (r *DashboardRouter) Init(group *gin.RouterGroup) {
	analytics := group.Group("/dashboard/analytics")
	analytics.Use(r.authMiddleware.Handle())
	{
		analytics.POST("/query", shared_pres.WrapHandler(r.analyticsHandler.ExecuteDynamicQuery))
	}

	adminDrill := group.Group("/admin/dashboard/drilldown")
	adminDrill.Use(r.authMiddleware.Handle(), middleware.RolesAuthorize(roleslug.Admin))
	{
		adminDrill.GET("/ai-events", shared_pres.WrapHandler(r.analyticsHandler.GetAIEventsDrillDown))
	}

	brandDrill := group.Group("/brand/dashboard/drilldown")
	brandDrill.Use(r.authMiddleware.Handle(), middleware.RolesAuthorize(roleslug.User, roleslug.Admin))
	{
		brandDrill.GET("/sample-lab-feedbacks", shared_pres.WrapHandler(r.analyticsHandler.GetSampleLabFeedbacksDrillDown))
	}
}
