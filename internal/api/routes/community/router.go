package community

import (
	"smart-wardrobe-be/internal/api/middleware"
	community_handler "smart-wardrobe-be/internal/modules/community/presentation/handler"
	shared_pres "smart-wardrobe-be/internal/shared/presentation"

	"github.com/gin-gonic/gin"
)

type CommunityRouter struct {
	postHandler        *community_handler.PostHandler
	interactionHandler *community_handler.PostInteractionHandler
	transferHandler    *community_handler.ItemTransferHandler
	authMiddleware     *middleware.AuthMiddleware
}

func NewRouter(
	postHandler *community_handler.PostHandler,
	interactionHandler *community_handler.PostInteractionHandler,
	transferHandler *community_handler.ItemTransferHandler,
	authMiddleware *middleware.AuthMiddleware,
) *CommunityRouter {
	return &CommunityRouter{
		postHandler:        postHandler,
		interactionHandler: interactionHandler,
		transferHandler:    transferHandler,
		authMiddleware:     authMiddleware,
	}
}

func (r *CommunityRouter) Init(group *gin.RouterGroup) {
	// Post - Public endpoints
	publicPosts := group.Group("/posts")
	{
		publicPosts.GET("", shared_pres.WrapHandler(r.postHandler.GetFeed))
		publicPosts.GET("/:postID", shared_pres.WrapHandler(r.postHandler.GetPostDetail))
	}

	// Post - Private endpoints (Authenticated)
	privatePosts := group.Group("/posts")
	privatePosts.Use(r.authMiddleware.Handle())
	{
		privatePosts.POST("", shared_pres.WrapHandler(r.postHandler.CreatePost))
		privatePosts.DELETE("/:postID", shared_pres.WrapHandler(r.postHandler.DeletePost))
		privatePosts.DELETE("/:postID/items", shared_pres.WrapHandler(r.postHandler.RemovePostItems))

		// Post Interaction (Likes & Comments)
		privatePosts.PUT("/:postID/like", shared_pres.WrapHandler(r.interactionHandler.TogglePostLike))
		privatePosts.POST("/:postID/comments", shared_pres.WrapHandler(r.interactionHandler.AddComment))
	}

	// Item Transfer (Authenticated)
	transfers := group.Group("/transfers")
	transfers.Use(r.authMiddleware.Handle())
	{
		transfers.GET("/pending", shared_pres.WrapHandler(r.transferHandler.GetPendingTransfers))
		transfers.POST("/:postItemID/accept", shared_pres.WrapHandler(r.transferHandler.AcceptTransfer))
		transfers.POST("/:postItemID/decline", shared_pres.WrapHandler(r.transferHandler.DeclineTransfer))
		transfers.POST("/:postItemID/mark-sold", shared_pres.WrapHandler(r.transferHandler.MarkPostItemSold))
	}
}
