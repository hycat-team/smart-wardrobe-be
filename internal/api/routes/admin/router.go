package admin

import (
	"smart-wardrobe-be/internal/api/middleware"
	identity_handler "smart-wardrobe-be/internal/modules/identity/presentation/handler"
	wardrobe_handler "smart-wardrobe-be/internal/modules/wardrobe/presentation/handler"
	"smart-wardrobe-be/internal/shared/domain/constants/identity/roleslug"
	shared_pres "smart-wardrobe-be/internal/shared/presentation"

	"github.com/gin-gonic/gin"
)

type AdminRouter struct {
	identityAdminHandler *identity_handler.AdminHandler
	wardrobeItemHandler  *wardrobe_handler.WardrobeItemHandler
	categoryHandler      *wardrobe_handler.CategoryHandler
	authMiddleware       *middleware.AuthMiddleware
}

func NewRouter(
	identityAdminHandler *identity_handler.AdminHandler,
	wardrobeItemHandler *wardrobe_handler.WardrobeItemHandler,
	categoryHandler *wardrobe_handler.CategoryHandler,
	authMiddleware *middleware.AuthMiddleware,
) *AdminRouter {
	return &AdminRouter{
		identityAdminHandler: identityAdminHandler,
		wardrobeItemHandler:  wardrobeItemHandler,
		categoryHandler:      categoryHandler,
		authMiddleware:       authMiddleware,
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

	// Phase 02 B2B2C rebuild: admin community/resale moderation routes are archived out of the MVP runtime.

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
