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
	publicApi := group.Group("/wardrobe-items")
	{
		publicApi.GET("/search", shared_pres.WrapHandler(r.wardrobeHandler.SearchWardrobeItems))
	}

	privateApi := group.Group("")
	privateApi.Use(r.authMiddleware.Handle())

	wardrobeApi := privateApi.Group("/wardrobe-items")
	{
		wardrobeApi.GET("/upload-signature", shared_pres.WrapHandler(r.wardrobeHandler.GetUploadSignature))
		wardrobeApi.GET("/:id", shared_pres.WrapHandler(r.wardrobeHandler.GetWardrobeItemByID))
		wardrobeApi.POST("/:id/clone", shared_pres.WrapHandler(r.wardrobeHandler.CloneWardrobeItem))
		wardrobeApi.POST("/catalog-init", shared_pres.WrapHandler(r.wardrobeHandler.InitClosetFromCatalog))
		wardrobeApi.POST("/batch-upload", shared_pres.WrapHandler(r.wardrobeHandler.BatchUploadWardrobeItems))
		wardrobeApi.PUT("/:id/manual-classify", shared_pres.WrapHandler(r.wardrobeHandler.ManualClassify))
	}

	aiApi := privateApi.Group("/ai")
	{
		aiApi.POST("/outfit-recommendations", shared_pres.WrapHandler(r.wardrobeHandler.RecommendOutfit))
		aiApi.POST("/chat/sessions", shared_pres.WrapHandler(r.wardrobeHandler.CreateChatSession))
		aiApi.GET("/chat/sessions", shared_pres.WrapHandler(r.wardrobeHandler.GetChatSessions))
		aiApi.GET("/chat/sessions/:contextID/messages", shared_pres.WrapHandler(r.wardrobeHandler.GetChatMessages))
		aiApi.PATCH("/chat/sessions/:contextID/archive", shared_pres.WrapHandler(r.wardrobeHandler.ArchiveChatSession))
		aiApi.POST("/chat/sessions/:contextID/messages/stream", shared_pres.WrapHandler(r.wardrobeHandler.StreamChatMessage))
	}

	meApi := privateApi.Group("/me/wardrobe-items")
	{
		meApi.GET("", shared_pres.WrapHandler(r.wardrobeHandler.GetWardrobeItems))
	}
}
