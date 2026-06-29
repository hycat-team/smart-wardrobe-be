package community

import (
	"smart-wardrobe-be/internal/api/middleware"
	community_handler "smart-wardrobe-be/internal/modules/community/presentation/handler"
	"smart-wardrobe-be/internal/shared/domain/constants/identity/roleslug"
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
	publicPosts.Use(r.authMiddleware.OptionalHandle())
	{
		publicPosts.GET("", shared_pres.WrapHandler(r.postHandler.GetFeed))
		publicPosts.GET("/:postPublicID", shared_pres.WrapHandler(r.postHandler.GetPostDetail))
		publicPosts.GET("/:postPublicID/comments", shared_pres.WrapHandler(r.postHandler.GetPostComments))
		publicPosts.GET("/:postPublicID/comments/:commentID/replies", shared_pres.WrapHandler(r.postHandler.GetCommentReplies))
		publicPosts.GET("/:postPublicID/likes", shared_pres.WrapHandler(r.postHandler.GetPostLikes))
	}

	// Post - Private endpoints (Authenticated)
	privatePosts := group.Group("/posts")
	privatePosts.Use(r.authMiddleware.Handle(), middleware.RolesAuthorize(roleslug.User))
	{
		privatePosts.GET("/upload-signature", shared_pres.WrapHandler(r.postHandler.GetUploadSignature))
		privatePosts.POST("", shared_pres.WrapHandler(r.postHandler.CreatePost))
		privatePosts.PUT("/:postPublicID", shared_pres.WrapHandler(r.postHandler.UpdatePost))
		privatePosts.DELETE("/:postPublicID", shared_pres.WrapHandler(r.postHandler.DeletePost))
		privatePosts.DELETE("/:postPublicID/items", shared_pres.WrapHandler(r.postHandler.RemovePostItems))

		// Post Interaction (Likes & Comments)
		privatePosts.PUT("/:postPublicID/like", shared_pres.WrapHandler(r.interactionHandler.TogglePostLike))
		privatePosts.POST("/:postPublicID/comments", shared_pres.WrapHandler(r.interactionHandler.AddComment))
		privatePosts.PUT("/:postPublicID/comments/:commentID", shared_pres.WrapHandler(r.interactionHandler.UpdateComment))
		privatePosts.DELETE("/:postPublicID/comments/:commentID", shared_pres.WrapHandler(r.interactionHandler.DeleteComment))
	}

	transfers := group.Group("/transfers")
	transfers.Use(r.authMiddleware.Handle(), middleware.RolesAuthorize(roleslug.User))

	// Transfer requests
	transfers.POST("/requests", shared_pres.WrapHandler(r.transferHandler.CreateTransferRequests))
	transfers.GET("/items/:postItemID/requests", shared_pres.WrapHandler(r.transferHandler.GetTransferRequestsForSeller))

	// Transfer bulk actions
	transfers.POST("/mark-sold", shared_pres.WrapHandler(r.transferHandler.MarkPostItemsSold))
	transfers.POST("/accept", shared_pres.WrapHandler(r.transferHandler.AcceptTransfers))
	transfers.POST("/decline", shared_pres.WrapHandler(r.transferHandler.DeclineTransfers))

	me := transfers.Group("/me")
	{
		me.GET("/pending", shared_pres.WrapHandler(r.transferHandler.GetPendingTransfers))
		me.GET("/posts", shared_pres.WrapHandler(r.transferHandler.GetSellerTransferPosts))
	}
}
