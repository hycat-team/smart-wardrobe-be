package router

import (
	aiexecutor_handler "smart-wardrobe-be/internal/modules/aiexecutor/presentation/handler"
	identity_contract "smart-wardrobe-be/internal/modules/identity/presentation/contract"
	"smart-wardrobe-be/internal/shared/domain/constants/roleslug"
	shared_pres "smart-wardrobe-be/internal/shared/presentation"
	shared_middleware "smart-wardrobe-be/internal/shared/presentation/middleware"

	"github.com/gin-gonic/gin"
)

type Router struct {
	adminHandler *aiexecutor_handler.AdminHandler
	auth         identity_contract.IAuthMiddleware
}

func NewRouter(
	adminHandler *aiexecutor_handler.AdminHandler,
	auth identity_contract.IAuthMiddleware,
) *Router {
	return &Router{
		adminHandler: adminHandler,
		auth:         auth,
	}
}

func (r *Router) Init(group *gin.RouterGroup) {
	adminAI := group.Group("/admin/ai")
	adminAI.Use(r.auth.Handle(), shared_middleware.RolesAuthorize(roleslug.Admin))
	{
		adminAI.GET("/pricing", shared_pres.WrapHandler(r.adminHandler.ListPricing))
		adminAI.PUT("/pricing/:id", shared_pres.WrapHandler(r.adminHandler.UpdatePricing))
		adminAI.GET("/usage", shared_pres.WrapHandler(r.adminHandler.GetUserUsage))
		adminAI.GET("/grants", shared_pres.WrapHandler(r.adminHandler.ListGrants))
		adminAI.POST("/grants", shared_pres.WrapHandler(r.adminHandler.CreateGrant))
	}
}
