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
	aiHandler           *fashion_handler.WardrobeAIHandler
	categoryHandler     *fashion_handler.CategoryHandler
	styleProfileHandler *fashion_handler.StyleProfileHandler
	auth                identity_contract.IAuthMiddleware
}

func NewRouter(
	aiHandler *fashion_handler.WardrobeAIHandler,
	categoryHandler *fashion_handler.CategoryHandler,
	styleProfileHandler *fashion_handler.StyleProfileHandler,
	auth identity_contract.IAuthMiddleware,
) *Router {
	return &Router{
		aiHandler:           aiHandler,
		categoryHandler:     categoryHandler,
		styleProfileHandler: styleProfileHandler,
		auth:                auth,
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
		// 033 US6: feedback on a previously returned recommendation.
		aiApi.POST("/outfit-recommendations/feedback", shared_pres.WrapHandler(r.aiHandler.SubmitRecommendationFeedback))
		// 033 US7: gap suggestions for a set of items the user already owns.
		// Never runs the LLM and never consumes the daily outfit quota.
		aiApi.POST("/outfit-completions", shared_pres.WrapHandler(r.aiHandler.CompleteOutfit))
		aiApi.POST("/chat/sessions", shared_pres.WrapHandler(r.aiHandler.CreateChatSession))
		aiApi.GET("/chat/sessions", shared_pres.WrapHandler(r.aiHandler.GetChatSessions))
		aiApi.GET("/chat/sessions/:contextID/messages", shared_pres.WrapHandler(r.aiHandler.GetChatMessages))
		aiApi.PATCH("/chat/sessions/:contextID/archive", shared_pres.WrapHandler(r.aiHandler.ArchiveChatSession))
		aiApi.DELETE("/chat/sessions/:contextID", shared_pres.WrapHandler(r.aiHandler.DeleteChatSession))
		aiApi.PATCH("/chat/sessions/:contextID", shared_pres.WrapHandler(r.aiHandler.UpdateChatSession))
		aiApi.POST("/chat/sessions/:contextID/messages/stream", shared_pres.WrapHandler(r.aiHandler.StreamChatMessage))
	}

	styleProfileApi := privateApi.Group("/me/style-profile")
	{
		styleProfileApi.GET("", shared_pres.WrapHandler(r.styleProfileHandler.GetStyleProfile))
		// 033 US5: cold-start suggestions derived from the active wardrobe.
		styleProfileApi.GET("/suggestions", shared_pres.WrapHandler(r.styleProfileHandler.GetStyleSuggestions))
		styleProfileApi.PUT("", shared_pres.WrapHandler(r.styleProfileHandler.UpdateStyleProfile))
		styleProfileApi.DELETE("", shared_pres.WrapHandler(r.styleProfileHandler.DeleteStyleProfile))
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
