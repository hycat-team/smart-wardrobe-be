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
	wardrobeItemHandler   *wardrobe_handler.WardrobeItemHandler
	categoryHandler       *wardrobe_handler.CategoryHandler
	authMiddleware        *middleware.AuthMiddleware
}

func NewRouter(
	identityAdminHandler *identity_handler.AdminHandler,
	communityAdminHandler *community_handler.AdminHandler,
	wardrobeItemHandler *wardrobe_handler.WardrobeItemHandler,
	categoryHandler *wardrobe_handler.CategoryHandler,
	authMiddleware *middleware.AuthMiddleware,
) *AdminRouter {
	return &AdminRouter{
		identityAdminHandler:  identityAdminHandler,
		communityAdminHandler: communityAdminHandler,
		wardrobeItemHandler:   wardrobeItemHandler,
		categoryHandler:       categoryHandler,
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
