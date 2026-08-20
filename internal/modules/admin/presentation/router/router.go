package router

import (
	admin_handler "smart-wardrobe-be/internal/modules/admin/presentation/handler"
	identity_contract "smart-wardrobe-be/internal/modules/identity/presentation/contract"
	"smart-wardrobe-be/internal/shared/domain/constants/roleslug"
	shared_pres "smart-wardrobe-be/internal/shared/presentation"
	shared_middleware "smart-wardrobe-be/internal/shared/presentation/middleware"

	"github.com/gin-gonic/gin"
)

type Router struct {
	auditHandler *admin_handler.AdminAuditHandler
	auth         identity_contract.IAuthMiddleware
}

func NewRouter(
	auditHandler *admin_handler.AdminAuditHandler,
	auth identity_contract.IAuthMiddleware,
) *Router {
	return &Router{
		auditHandler: auditHandler,
		auth:         auth,
	}
}

func (r *Router) Init(group *gin.RouterGroup) {
	adminAudit := group.Group("/admin/audit-logs")
	adminAudit.Use(r.auth.Handle(), shared_middleware.RolesAuthorize(roleslug.Admin))
	{
		adminAudit.GET("", shared_pres.WrapHandler(r.auditHandler.GetAuditLogs))
	}
}
