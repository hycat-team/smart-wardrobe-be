package wardrobe

import (
	"smart-wardrobe-be/internal/api/middleware"
	wardrobe_handler "smart-wardrobe-be/internal/modules/wardrobe/presentation/handler"
	"smart-wardrobe-be/internal/shared/domain/constants/identity/roleslug"
	shared_pres "smart-wardrobe-be/internal/shared/presentation"

	"github.com/gin-gonic/gin"
)

type WardrobeRouter struct {
	itemHandler    *wardrobe_handler.WardrobeItemHandler
	authMiddleware *middleware.AuthMiddleware
}

func NewRouter(itemHandler *wardrobe_handler.WardrobeItemHandler, m *middleware.AuthMiddleware) *WardrobeRouter {
	return &WardrobeRouter{
		itemHandler:    itemHandler,
		authMiddleware: m,
	}
}

func (r *WardrobeRouter) Init(group *gin.RouterGroup) {
	publicApi := group.Group("/system-catalog/wardrobe-items")
	{
		publicApi.GET("", shared_pres.WrapHandler(r.itemHandler.GetSystemCatalogWardrobeItems))
	}

	privateApi := group.Group("")
	privateApi.Use(r.authMiddleware.Handle(), middleware.RolesAuthorize(roleslug.User))
	wardrobeApi := privateApi.Group("/wardrobe-items")
	{
		wardrobeApi.GET("/upload-signature", shared_pres.WrapHandler(r.itemHandler.GetUploadSignature))
		wardrobeApi.DELETE("/bulk", shared_pres.WrapHandler(r.itemHandler.DeleteWardrobeItemsBulk))
		wardrobeApi.DELETE("/locked", shared_pres.WrapHandler(r.itemHandler.DeleteLockedWardrobeItems))
		wardrobeApi.GET("/:id", shared_pres.WrapHandler(r.itemHandler.GetWardrobeItemByID))
		wardrobeApi.POST("/:id/clone", shared_pres.WrapHandler(r.itemHandler.CloneWardrobeItem))
		wardrobeApi.POST("/catalog-init", shared_pres.WrapHandler(r.itemHandler.InitClosetFromCatalog))
		wardrobeApi.POST("/batch-upload", shared_pres.WrapHandler(r.itemHandler.BatchUploadWardrobeItems))
		wardrobeApi.POST("/:id/retry-analysis", shared_pres.WrapHandler(r.itemHandler.RetryWardrobeAnalysis))
		wardrobeApi.PUT("/:id/manual-classify", shared_pres.WrapHandler(r.itemHandler.ManualClassify))
	}

	meApi := privateApi.Group("/me/wardrobe-items")
	{
		meApi.GET("", shared_pres.WrapHandler(r.itemHandler.GetWardrobeItems))
		meApi.GET("/stats", shared_pres.WrapHandler(r.itemHandler.GetWardrobeStats))
	}
}
