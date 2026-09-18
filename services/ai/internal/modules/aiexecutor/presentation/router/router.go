package router

import (
	identity_contract "smart-wardrobe-be/services/account/contract/identity"
	aiexecutor_handler "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/presentation/handler"
	"smart-wardrobe-be/shared/domain/constants/roleslug"
	shared_pres "smart-wardrobe-be/shared/presentation"
	shared_middleware "smart-wardrobe-be/shared/presentation/middleware"

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
