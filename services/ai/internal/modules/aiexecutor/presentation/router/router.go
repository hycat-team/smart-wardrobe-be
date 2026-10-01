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
	adminHandler           *aiexecutor_handler.AdminHandler
	costPolicyAdminHandler *aiexecutor_handler.CostPolicyAdminHandler
	auth                   identity_contract.IAuthMiddleware
}

func NewRouter(
	adminHandler *aiexecutor_handler.AdminHandler,
	costPolicyAdminHandler *aiexecutor_handler.CostPolicyAdminHandler,
	auth identity_contract.IAuthMiddleware,
) *Router {
	return &Router{
		adminHandler:           adminHandler,
		costPolicyAdminHandler: costPolicyAdminHandler,
		auth:                   auth,
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
		// 024 T070: POST /grants is removed. It could never succeed — it left
		// policy_snapshot unset on a NOT NULL column — and it was a second
		// writer on a table the AI service must own alone (FR-035). Granting a
		// plan goes through the Account service and the ordinary plan-change
		// event.
		// Gate 2 (ai_cost_policy_operations). Stays inside the existing group so
		// it inherits auth.Handle() + RolesAuthorize(roleslug.Admin) (024 FR-004b).
		adminAI.GET("/cost-policies", shared_pres.WrapHandler(r.costPolicyAdminHandler.ListCostPolicies))
		adminAI.PUT("/cost-policies/:id/operations/:operation", shared_pres.WrapHandler(r.costPolicyAdminHandler.UpdateCostPolicyOperation))
	}
}
