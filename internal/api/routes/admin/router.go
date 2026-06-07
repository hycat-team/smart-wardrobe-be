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
		adminUsers.GET("", shared_pres.WrapHandler(r.identityAdminHandler.GetUsers))
		adminUsers.PATCH("/:id/status", shared_pres.WrapHandler(r.identityAdminHandler.UpdateUserStatus))
	}

	adminPosts := admin.Group("/posts")
	{
		adminPosts.GET("", shared_pres.WrapHandler(r.communityAdminHandler.GetPosts))
		adminPosts.DELETE("/:postPublicID", shared_pres.WrapHandler(r.communityAdminHandler.DeletePost))
		adminPosts.PATCH("/:postPublicID/restore", shared_pres.WrapHandler(r.communityAdminHandler.RestorePost))
	}

	adminComments := admin.Group("/comments")
	{
		adminComments.DELETE("/:commentID", shared_pres.WrapHandler(r.communityAdminHandler.DeleteComment))
		adminComments.PATCH("/:commentID/restore", shared_pres.WrapHandler(r.communityAdminHandler.RestoreComment))
	}

	adminPostItems := admin.Group("/post-items")
	{
		adminPostItems.GET("", shared_pres.WrapHandler(r.communityAdminHandler.GetPostItems))
		adminPostItems.PATCH("/:postItemID/hide", shared_pres.WrapHandler(r.communityAdminHandler.HidePostItem))
		adminPostItems.DELETE("/:postItemID", shared_pres.WrapHandler(r.communityAdminHandler.DeletePostItem))
	}

	adminWardrobe := admin.Group("/wardrobe-items")
	{
		adminWardrobe.GET("", shared_pres.WrapHandler(r.wardrobeHandler.GetCatalogItemsAdmin))
		adminWardrobe.PUT("/:id", shared_pres.WrapHandler(r.wardrobeHandler.UpdateCatalogItemAdmin))
		adminWardrobe.DELETE("/:id", shared_pres.WrapHandler(r.wardrobeHandler.DeleteCatalogItemAdmin))
		adminWardrobe.GET("/upload-signature", shared_pres.WrapHandler(r.wardrobeHandler.GetUploadSignature))
		adminWardrobe.POST("/batch-upload", shared_pres.WrapHandler(r.wardrobeHandler.BatchUploadWardrobeItems))
	}
}
