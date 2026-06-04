package outfit

import (
	"smart-wardrobe-be/internal/api/middleware"
	"smart-wardrobe-be/internal/modules/wardrobe/presentation/handler"
	shared_pres "smart-wardrobe-be/internal/shared/presentation"

	"github.com/gin-gonic/gin"
)

type OutfitRouter struct {
	outfitHandler  *handler.OutfitHandler
	authMiddleware *middleware.AuthMiddleware
}

func NewRouter(
	outfitHandler *handler.OutfitHandler,
	authMiddleware *middleware.AuthMiddleware,
) *OutfitRouter {
	return &OutfitRouter{
		outfitHandler:  outfitHandler,
		authMiddleware: authMiddleware,
	}
}

func (r *OutfitRouter) Init(parentGroup *gin.RouterGroup) {
	privateApi := parentGroup.Group("")
	privateApi.Use(r.authMiddleware.Handle())

	outfitApi := privateApi.Group("/outfits")
	{
		outfitApi.GET("/upload-signature", shared_pres.WrapHandler(r.outfitHandler.GetUploadSignature))
		outfitApi.POST("", shared_pres.WrapHandler(r.outfitHandler.SaveOutfit))
		outfitApi.PUT("/:id", shared_pres.WrapHandler(r.outfitHandler.UpdateOutfit))
		outfitApi.GET("/:id", shared_pres.WrapHandler(r.outfitHandler.GetOutfitByID))
		outfitApi.DELETE("/:id", shared_pres.WrapHandler(r.outfitHandler.DeleteOutfit))
	}

	meApi := privateApi.Group("/me/outfits")
	{
		meApi.GET("", shared_pres.WrapHandler(r.outfitHandler.GetOutfits))
	}
}
