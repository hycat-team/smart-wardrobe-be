package router

import (
	identity_contract "smart-wardrobe-be/services/account/contract/identity"
	community_handler "smart-wardrobe-be/services/commerce/internal/modules/community/presentation/handler"
	"smart-wardrobe-be/shared/domain/constants/roleslug"
	shared_pres "smart-wardrobe-be/shared/presentation"
	shared_middleware "smart-wardrobe-be/shared/presentation/middleware"

	"github.com/gin-gonic/gin"
)

type Router struct {
	postHandler        *community_handler.PostHandler
	interactionHandler *community_handler.InteractionHandler
	followHandler      *community_handler.FollowHandler
	profileHandler     *community_handler.ProfileHandler
	searchHandler      *community_handler.SearchHandler
	adminHandler       *community_handler.AdminHandler
	auth               identity_contract.IAuthMiddleware
}

func NewRouter(
	postHandler *community_handler.PostHandler,
	interactionHandler *community_handler.InteractionHandler,
	followHandler *community_handler.FollowHandler,
	profileHandler *community_handler.ProfileHandler,
	searchHandler *community_handler.SearchHandler,
	adminHandler *community_handler.AdminHandler,
	auth identity_contract.IAuthMiddleware,
) *Router {
	return &Router{
		postHandler:        postHandler,
		interactionHandler: interactionHandler,
		followHandler:      followHandler,
		profileHandler:     profileHandler,
		searchHandler:      searchHandler,
		adminHandler:       adminHandler,
		auth:               auth,
	}
}

func (r *Router) Init(group *gin.RouterGroup) {
	// Public read endpoints (optional auth so viewer flags can be computed).
	publicPosts := group.Group("/posts")
	publicPosts.Use(r.auth.OptionalHandle())
	{
		publicPosts.GET("", shared_pres.WrapHandler(r.postHandler.GetFeed))
		publicPosts.GET("/:postPublicID", shared_pres.WrapHandler(r.postHandler.GetPostDetail))
		publicPosts.GET("/:postPublicID/comments", shared_pres.WrapHandler(r.interactionHandler.GetPostComments))
		publicPosts.GET("/:postPublicID/comments/:commentID/replies", shared_pres.WrapHandler(r.interactionHandler.GetCommentReplies))
		publicPosts.GET("/:postPublicID/likes", shared_pres.WrapHandler(r.interactionHandler.GetPostLikes))
	}

	// Authenticated write endpoints.
	privatePosts := group.Group("/posts")
	privatePosts.Use(r.auth.Handle(), shared_middleware.RolesAuthorize(roleslug.User))
	{
		privatePosts.GET("/upload-signature", shared_pres.WrapHandler(r.postHandler.GetUploadSignature))
		privatePosts.POST("", shared_pres.WrapHandler(r.postHandler.CreatePost))
		privatePosts.PUT("/:postPublicID", shared_pres.WrapHandler(r.postHandler.UpdatePost))
		privatePosts.DELETE("/:postPublicID", shared_pres.WrapHandler(r.postHandler.DeletePost))
		privatePosts.PUT("/:postPublicID/like", shared_pres.WrapHandler(r.interactionHandler.TogglePostLike))
		privatePosts.POST("/:postPublicID/comments", shared_pres.WrapHandler(r.interactionHandler.AddComment))
		privatePosts.PUT("/:postPublicID/comments/:commentID", shared_pres.WrapHandler(r.interactionHandler.UpdateComment))
		privatePosts.DELETE("/:postPublicID/comments/:commentID", shared_pres.WrapHandler(r.interactionHandler.DeleteComment))
	}

	// Public profile endpoints (optional auth for viewer flags).
	publicUsers := group.Group("/users")
	publicUsers.Use(r.auth.OptionalHandle())
	{
		publicUsers.GET("/:username", shared_pres.WrapHandler(r.profileHandler.GetPublicProfile))
		publicUsers.GET("/:username/posts", shared_pres.WrapHandler(r.profileHandler.GetUserPosts))
		publicUsers.GET("/:username/follows", shared_pres.WrapHandler(r.followHandler.ListFollows))
	}

	privateUsers := group.Group("/users")
	privateUsers.Use(r.auth.Handle(), shared_middleware.RolesAuthorize(roleslug.User))
	{
		privateUsers.PUT("/:username/follow", shared_pres.WrapHandler(r.followHandler.ToggleFollow))
	}

	publicSearch := group.Group("/search")
	publicSearch.Use(r.auth.OptionalHandle())
	{
		publicSearch.GET("", shared_pres.WrapHandler(r.searchHandler.Search))
	}

	admin := group.Group("/admin")
	admin.Use(r.auth.Handle(), shared_middleware.RolesAuthorize(roleslug.Admin))
	{
		adminPosts := admin.Group("/posts")
		adminPosts.GET("", shared_pres.WrapHandler(r.adminHandler.ListPosts))
		adminPosts.PATCH("/:postID/hide", shared_pres.WrapHandler(r.adminHandler.HidePost))
		adminPosts.PATCH("/:postID/restore", shared_pres.WrapHandler(r.adminHandler.RestorePost))
		adminPosts.DELETE("/:postID", shared_pres.WrapHandler(r.adminHandler.DeletePost))

		adminComments := admin.Group("/comments")
		adminComments.GET("", shared_pres.WrapHandler(r.adminHandler.ListComments))
		adminComments.PATCH("/:commentID/hide", shared_pres.WrapHandler(r.adminHandler.HideComment))
		adminComments.PATCH("/:commentID/restore", shared_pres.WrapHandler(r.adminHandler.RestoreComment))
		adminComments.DELETE("/:commentID", shared_pres.WrapHandler(r.adminHandler.DeleteComment))
	}
}
