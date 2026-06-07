package admin

import (
	"smart-wardrobe-be/internal/api/middleware"
	community_handler "smart-wardrobe-be/internal/modules/community/presentation/handler"
	identity_handler "smart-wardrobe-be/internal/modules/identity/presentation/handler"
	wardrobe_handler "smart-wardrobe-be/internal/modules/wardrobe/presentation/handler"
	"smart-wardrobe-be/internal/shared/domain/constants/roleslug"
	shared_pres "smart-wardrobe-be/internal/shared/presentation"

	"github.com/gin-gonic/gin"
)

type AdminRouter struct {
	identityAdminHandler  *identity_handler.AdminHandler
	communityAdminHandler *community_handler.AdminHandler
	wardrobeHandler       *wardrobe_handler.WardrobeHandler
	authMiddleware        *middleware.AuthMiddleware
}

func NewRouter(
	identityAdminHandler *identity_handler.AdminHandler,
	communityAdminHandler *community_handler.AdminHandler,
	wardrobeHandler *wardrobe_handler.WardrobeHandler,
	authMiddleware *middleware.AuthMiddleware,
) *AdminRouter {
	return &AdminRouter{
		identityAdminHandler:  identityAdminHandler,
		communityAdminHandler: communityAdminHandler,
		wardrobeHandler:       wardrobeHandler,
		authMiddleware:        authMiddleware,
	}
}

func (r *AdminRouter) Init(group *gin.RouterGroup) {
	admin := group.Group("/admin")
	admin.Use(r.authMiddleware.Handle(), middleware.RolesAuthorize(roleslug.Admin))

	adminUsers := admin.Group("/users")
	{
		adminUsers.PATCH("/:id/status", shared_pres.WrapHandler(r.identityAdminHandler.UpdateUserStatus))
	}

	adminCommunity := admin.Group("/community")
	{
		adminCommunity.DELETE("/posts/:postPublicID", shared_pres.WrapHandler(r.communityAdminHandler.DeletePost))
		adminCommunity.DELETE("/comments/:commentID", shared_pres.WrapHandler(r.communityAdminHandler.DeleteComment))
		adminCommunity.PATCH("/post-items/:postItemID/hide", shared_pres.WrapHandler(r.communityAdminHandler.HidePostItem))
		adminCommunity.DELETE("/post-items/:postItemID", shared_pres.WrapHandler(r.communityAdminHandler.DeletePostItem))
	}

	adminWardrobe := admin.Group("/wardrobe-items")
	{
		adminWardrobe.GET("/upload-signature", shared_pres.WrapHandler(r.wardrobeHandler.GetUploadSignature))
		adminWardrobe.POST("/batch-upload", shared_pres.WrapHandler(r.wardrobeHandler.BatchUploadWardrobeItems))
	}
}
