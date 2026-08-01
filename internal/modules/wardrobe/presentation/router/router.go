package router

import (
	identity_contract "smart-wardrobe-be/internal/modules/identity/presentation/contract"
	wardrobe_handler "smart-wardrobe-be/internal/modules/wardrobe/presentation/handler"
	"smart-wardrobe-be/internal/shared/domain/constants/roleslug"
	shared_pres "smart-wardrobe-be/internal/shared/presentation"
	shared_middleware "smart-wardrobe-be/internal/shared/presentation/middleware"

	"github.com/gin-gonic/gin"
)

type Router struct {
	itemHandler     *wardrobe_handler.WardrobeItemHandler
	outfitHandler   *wardrobe_handler.OutfitHandler
	categoryHandler *wardrobe_handler.CategoryHandler
	auth            identity_contract.IAuthMiddleware
}

func NewRouter(
	itemHandler *wardrobe_handler.WardrobeItemHandler,
	outfitHandler *wardrobe_handler.OutfitHandler,
	categoryHandler *wardrobe_handler.CategoryHandler,
	auth identity_contract.IAuthMiddleware,
) *Router {
	return &Router{
		itemHandler:     itemHandler,
		outfitHandler:   outfitHandler,
		categoryHandler: categoryHandler,
		auth:            auth,
	}
}

func (r *Router) Init(group *gin.RouterGroup) {
	publicApi := group.Group("/system-catalog/wardrobe-items")
	{
		publicApi.GET("", shared_pres.WrapHandler(r.itemHandler.GetSystemCatalogWardrobeItems))
	}

	categoryApi := group.Group("/categories")
	{
		categoryApi.GET("", shared_pres.WrapHandler(r.categoryHandler.GetCategories))
	}

	privateApi := group.Group("")
	privateApi.Use(r.auth.Handle(), shared_middleware.RolesAuthorize(roleslug.User))

	wardrobeApi := privateApi.Group("/wardrobe-items")
	{
		wardrobeApi.GET("/upload-signature", shared_pres.WrapHandler(r.itemHandler.GetUploadSignature))
		wardrobeApi.DELETE("/bulk", shared_pres.WrapHandler(r.itemHandler.DeleteWardrobeItemsBulk))
		wardrobeApi.DELETE("/locked", shared_pres.WrapHandler(r.itemHandler.DeleteLockedWardrobeItems))
		wardrobeApi.GET("/:id", shared_pres.WrapHandler(r.itemHandler.GetWardrobeItemByID))
		wardrobeApi.POST("/:id/clone", shared_pres.WrapHandler(r.itemHandler.CloneWardrobeItem))
		wardrobeApi.POST("/catalog-init", shared_pres.WrapHandler(r.itemHandler.InitClosetFromCatalog))
		wardrobeApi.POST("/batch-upload", shared_pres.WrapHandler(r.itemHandler.BatchUploadWardrobeItems))
		wardrobeApi.GET("/tasks/:taskId/sse", shared_pres.WrapHandler(r.itemHandler.SubscribeTaskSSE))
		wardrobeApi.POST("/:id/retry-analysis", shared_pres.WrapHandler(r.itemHandler.RetryWardrobeAnalysis))
		wardrobeApi.PUT("/:id/manual-classify", shared_pres.WrapHandler(r.itemHandler.ManualClassify))
	}

	meApi := privateApi.Group("/me/wardrobe-items")
	{
		meApi.GET("", shared_pres.WrapHandler(r.itemHandler.GetWardrobeItems))
		meApi.GET("/stats", shared_pres.WrapHandler(r.itemHandler.GetWardrobeStats))
	}

	outfitApi := privateApi.Group("/outfits")
	{
		outfitApi.GET("/upload-signature", shared_pres.WrapHandler(r.outfitHandler.GetUploadSignature))
		outfitApi.POST("", shared_pres.WrapHandler(r.outfitHandler.SaveOutfit))
		outfitApi.PUT("/:id", shared_pres.WrapHandler(r.outfitHandler.UpdateOutfit))
		outfitApi.GET("/:id", shared_pres.WrapHandler(r.outfitHandler.GetOutfitByID))
		outfitApi.DELETE("/:id", shared_pres.WrapHandler(r.outfitHandler.DeleteOutfit))
	}

	meOutfits := privateApi.Group("/me/outfits")
	{
		meOutfits.GET("", shared_pres.WrapHandler(r.outfitHandler.GetOutfits))
	}

	admin := group.Group("/admin")
	admin.Use(r.auth.Handle(), shared_middleware.RolesAuthorize(roleslug.Admin))

	adminWardrobe := admin.Group("/wardrobe-items")
	{
		adminWardrobe.GET("", shared_pres.WrapHandler(r.itemHandler.GetCatalogItemsAdmin))
		adminWardrobe.PUT("/:id", shared_pres.WrapHandler(r.itemHandler.UpdateCatalogItemAdmin))
		adminWardrobe.DELETE("/:id", shared_pres.WrapHandler(r.itemHandler.DeleteCatalogItemAdmin))
		adminWardrobe.GET("/upload-signature", shared_pres.WrapHandler(r.itemHandler.GetUploadSignature))
		adminWardrobe.POST("/batch-upload", shared_pres.WrapHandler(r.itemHandler.BatchUploadWardrobeItems))
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
