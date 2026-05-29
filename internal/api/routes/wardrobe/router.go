package wardrobe

import (
	"smart-wardrobe-be/internal/api/middleware"
	wardrobe_handler "smart-wardrobe-be/internal/modules/wardrobe/presentation/handler"
	shared_pres "smart-wardrobe-be/internal/shared/presentation"

	"github.com/gin-gonic/gin"
)

type WardrobeRouter struct {
	wardrobeHandler *wardrobe_handler.WardrobeHandler
	authMiddleware  *middleware.AuthMiddleware
}

func NewRouter(h *wardrobe_handler.WardrobeHandler, m *middleware.AuthMiddleware) *WardrobeRouter {
	return &WardrobeRouter{
		wardrobeHandler: h,
		authMiddleware:  m,
	}
}

func (r *WardrobeRouter) Init(group *gin.RouterGroup) {
	wardrobeApi := group.Group("/wardrobe-items")
	wardrobeApi.Use(r.authMiddleware.Handle())
	{
		wardrobeApi.GET("/upload-signature", shared_pres.WrapHandler(r.wardrobeHandler.GetUploadSignature))
		wardrobeApi.POST("", shared_pres.WrapHandler(r.wardrobeHandler.CreateWardrobeItem))
	}
}
