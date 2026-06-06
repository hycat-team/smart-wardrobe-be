package community

import (
	"smart-wardrobe-be/internal/api/middleware"
	community_handler "smart-wardrobe-be/internal/modules/community/presentation/handler"
	"smart-wardrobe-be/internal/shared/domain/constants/roleslug"
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
		publicPosts.GET("/:postID", shared_pres.WrapHandler(r.postHandler.GetPostDetail))
	}

	// Post - Private endpoints (Authenticated)
	privatePosts := group.Group("/posts")
	privatePosts.Use(r.authMiddleware.Handle(), middleware.RolesAuthorize(roleslug.Member))
	{
		privatePosts.GET("/upload-signature", shared_pres.WrapHandler(r.postHandler.GetUploadSignature))
		privatePosts.POST("", shared_pres.WrapHandler(r.postHandler.CreatePost))
		privatePosts.DELETE("/:postID", shared_pres.WrapHandler(r.postHandler.DeletePost))
		privatePosts.DELETE("/:postID/items", shared_pres.WrapHandler(r.postHandler.RemovePostItems))

		// Post Interaction (Likes & Comments)
		privatePosts.PUT("/:postID/like", shared_pres.WrapHandler(r.interactionHandler.TogglePostLike))
		privatePosts.POST("/:postID/comments", shared_pres.WrapHandler(r.interactionHandler.AddComment))
		privatePosts.PUT("/:postID/comments/:commentID", shared_pres.WrapHandler(r.interactionHandler.UpdateComment))
		privatePosts.DELETE("/:postID/comments/:commentID", shared_pres.WrapHandler(r.interactionHandler.DeleteComment))
	}

	community := group.Group("/community")
	community.Use(r.authMiddleware.Handle(), middleware.RolesAuthorize(roleslug.Member))
	me := community.Group("/me")
	{
		me.GET("/pending-transfers", shared_pres.WrapHandler(r.transferHandler.GetPendingTransfers))
		me.GET("/transfer-posts", shared_pres.WrapHandler(r.transferHandler.GetSellerTransferPosts))
	}

	postItems := community.Group("/post-items")
	{
		postItems.POST("/:postItemID/accept", shared_pres.WrapHandler(r.transferHandler.AcceptTransfer))
		postItems.POST("/:postItemID/decline", shared_pres.WrapHandler(r.transferHandler.DeclineTransfer))
		postItems.POST("/:postItemID/mark-sold", shared_pres.WrapHandler(r.transferHandler.MarkPostItemSold))
	}
}
