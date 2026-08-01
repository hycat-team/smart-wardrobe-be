package router

import (
	fashion_handler "smart-wardrobe-be/internal/modules/fashion/presentation/handler"
	identity_contract "smart-wardrobe-be/internal/modules/identity/presentation/contract"
	"smart-wardrobe-be/internal/shared/domain/constants/roleslug"
	shared_pres "smart-wardrobe-be/internal/shared/presentation"
	shared_middleware "smart-wardrobe-be/internal/shared/presentation/middleware"

	"github.com/gin-gonic/gin"
)

type Router struct {
	aiHandler *fashion_handler.WardrobeAIHandler
	auth      identity_contract.IAuthMiddleware
}

func NewRouter(
	aiHandler *fashion_handler.WardrobeAIHandler,
	auth identity_contract.IAuthMiddleware,
) *Router {
	return &Router{
		aiHandler: aiHandler,
		auth:      auth,
	}
}

func (r *Router) Init(group *gin.RouterGroup) {
	privateApi := group.Group("")
	privateApi.Use(r.auth.Handle(), shared_middleware.RolesAuthorize(roleslug.User))

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
