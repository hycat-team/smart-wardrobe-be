package router

import (
	identity_contract "smart-wardrobe-be/services/account/contract/identity"
	fashion_handler "smart-wardrobe-be/services/commerce/internal/modules/fashion/presentation/handler"
	"smart-wardrobe-be/shared/domain/constants/roleslug"
	shared_pres "smart-wardrobe-be/shared/presentation"
	shared_middleware "smart-wardrobe-be/shared/presentation/middleware"

	"github.com/gin-gonic/gin"
)

type Router struct {
	aiHandler       *fashion_handler.WardrobeAIHandler
	categoryHandler *fashion_handler.CategoryHandler
	auth            identity_contract.IAuthMiddleware
}

func NewRouter(
	aiHandler *fashion_handler.WardrobeAIHandler,
	categoryHandler *fashion_handler.CategoryHandler,
	auth identity_contract.IAuthMiddleware,
) *Router {
	return &Router{
		aiHandler:       aiHandler,
		categoryHandler: categoryHandler,
		auth:            auth,
	}
}

func (r *Router) Init(group *gin.RouterGroup) {
	publicApi := group.Group("")
	{
		publicApi.GET("/categories", shared_pres.WrapHandler(r.categoryHandler.GetCategories))
	}

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

	admin := group.Group("/admin")
	admin.Use(r.auth.Handle(), shared_middleware.RolesAuthorize(roleslug.Admin))

	adminCategories := admin.Group("/categories")
	{
		adminCategories.GET("", shared_pres.WrapHandler(r.categoryHandler.GetCategoriesAdmin))
		adminCategories.GET("/:id", shared_pres.WrapHandler(r.categoryHandler.GetCategoryByIDAdmin))
		adminCategories.POST("", shared_pres.WrapHandler(r.categoryHandler.CreateCategoryAdmin))
		adminCategories.PUT("/:id", shared_pres.WrapHandler(r.categoryHandler.UpdateCategoryAdmin))
		adminCategories.DELETE("/:id", shared_pres.WrapHandler(r.categoryHandler.DeleteCategoryAdmin))
	}
}
