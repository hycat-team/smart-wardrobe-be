package fashion

import (
	"smart-wardrobe-be/internal/api/middleware"
	fashion_handler "smart-wardrobe-be/internal/modules/fashion/presentation/handler"
	"smart-wardrobe-be/internal/shared/domain/constants/roleslug"
	shared_pres "smart-wardrobe-be/internal/shared/presentation"

	"github.com/gin-gonic/gin"
)

type FashionRouter struct {
	aiHandler      *fashion_handler.WardrobeAIHandler
	authMiddleware *middleware.AuthMiddleware
}

func NewRouter(aiHandler *fashion_handler.WardrobeAIHandler, m *middleware.AuthMiddleware) *FashionRouter {
	return &FashionRouter{
		aiHandler:      aiHandler,
		authMiddleware: m,
	}
}

func (r *FashionRouter) Init(group *gin.RouterGroup) {
	privateApi := group.Group("")
	privateApi.Use(r.authMiddleware.Handle(), middleware.RolesAuthorize(roleslug.User))

	aiApi := privateApi.Group("/ai")
	{
		aiApi.POST("/outfit-recommendations", shared_pres.WrapHandler(r.aiHandler.RecommendOutfit))
		aiApi.POST("/chat/sessions", shared_pres.WrapHandler(r.aiHandler.CreateChatSession))
		aiApi.GET("/chat/sessions", shared_pres.WrapHandler(r.aiHandler.GetChatSessions))
		aiApi.GET("/chat/sessions/:contextID/messages", shared_pres.WrapHandler(r.aiHandler.GetChatMessages))
		aiApi.PATCH("/chat/sessions/:contextID/archive", shared_pres.WrapHandler(r.aiHandler.ArchiveChatSession))
		aiApi.DELETE("/chat/sessions/:contextID", shared_pres.WrapHandler(r.aiHandler.DeleteChatSession))
		aiApi.PATCH("/chat/sessions/:contextID", shared_pres.WrapHandler(r.aiHandler.UpdateChatSession))
		aiApi.POST("/chat/sessions/:contextID/messages/stream", shared_pres.WrapHandler(r.aiHandler.StreamChatMessage))
	}
}
