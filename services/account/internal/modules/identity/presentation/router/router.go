package router

import (
	identity_contract "smart-wardrobe-be/services/account/contract/identity"
	identity_handler "smart-wardrobe-be/services/account/internal/modules/identity/presentation/handler"
	"smart-wardrobe-be/shared/domain/constants/roleslug"
	shared_pres "smart-wardrobe-be/shared/presentation"
	shared_middleware "smart-wardrobe-be/shared/presentation/middleware"

	"github.com/gin-gonic/gin"
)

type Router struct {
	authHandler  *identity_handler.AuthHandler
	meHandler    *identity_handler.MeHandler
	adminHandler *identity_handler.AdminHandler
	auth         identity_contract.IAuthMiddleware
	loginLimit   *shared_middleware.LoginRateLimitMiddleware
}

func NewRouter(
	authHandler *identity_handler.AuthHandler,
	meHandler *identity_handler.MeHandler,
	adminHandler *identity_handler.AdminHandler,
	auth identity_contract.IAuthMiddleware,
	loginLimit *shared_middleware.LoginRateLimitMiddleware,
) *Router {
	return &Router{
		authHandler:  authHandler,
		meHandler:    meHandler,
		adminHandler: adminHandler,
		auth:         auth,
		loginLimit:   loginLimit,
	}
}

func (r *Router) Init(group *gin.RouterGroup) {
	authApi := group.Group("/auth")
	authApi.Use(r.loginLimit.Handle())
	{
		authApi.POST("/register", shared_pres.WrapHandler(r.authHandler.Register))
		authApi.POST("/register/confirm-otp", shared_pres.WrapHandler(r.authHandler.ConfirmRegisterOtp))
		authApi.POST("/register/resend-otp", shared_pres.WrapHandler(r.authHandler.ResendRegisterOtp))
		authApi.POST("/login", shared_pres.WrapHandler(r.authHandler.Login))
		authApi.POST("/google", shared_pres.WrapHandler(r.authHandler.GoogleLogin))
		authApi.GET("/google", shared_pres.WrapHandler(r.authHandler.GoogleStart))
		authApi.GET("/google/callback", shared_pres.WrapHandler(r.authHandler.GoogleCallback))
		authApi.POST("/refresh-token", shared_pres.WrapHandler(r.authHandler.RefreshToken))
		authApi.POST("/forgot-password", shared_pres.WrapHandler(r.authHandler.ForgotPassword))
		authApi.POST("/forgot-password/confirm-otp", shared_pres.WrapHandler(r.authHandler.ConfirmForgotPasswordOtp))
		authApi.POST("/forgot-password/resend-otp", shared_pres.WrapHandler(r.authHandler.ResendForgotPasswordOtp))
		authApi.POST("/reset-password", shared_pres.WrapHandler(r.authHandler.ResetPassword))
	}

	privateAuth := authApi.Group("")
	privateAuth.Use(r.auth.Handle())
	{
		privateAuth.POST("/logout", shared_pres.WrapHandler(r.authHandler.Logout))
	}

	meApi := group.Group("/me")
	meApi.Use(r.auth.Handle())
	{
		meApi.GET("", shared_pres.WrapHandler(r.meHandler.GetCurrentUser))
		meApi.PUT("", shared_pres.WrapHandler(r.meHandler.UpdateCurrentUser))
		meApi.PUT("/body-profile", shared_pres.WrapHandler(r.meHandler.UpdateBodyProfile))
		meApi.PUT("/change-password", shared_pres.WrapHandler(r.meHandler.ChangePassword))
		meApi.GET("/avatar-signature", shared_pres.WrapHandler(r.meHandler.GetAvatarSignature))
		meApi.PUT("/avatar", shared_pres.WrapHandler(r.meHandler.UpdateAvatar))
	}

	adminUsers := group.Group("/admin/users")
	adminUsers.Use(r.auth.Handle(), shared_middleware.RolesAuthorize(roleslug.Admin))
	{
		adminUsers.GET("", shared_pres.WrapHandler(r.adminHandler.GetUsers))
		adminUsers.POST("", shared_pres.WrapHandler(r.adminHandler.CreateUser))
		adminUsers.GET("/:id", shared_pres.WrapHandler(r.adminHandler.GetUserDetail))
		adminUsers.PATCH("/:id/role", shared_pres.WrapHandler(r.adminHandler.UpdateUserRole))
		adminUsers.PATCH("/:id/status", shared_pres.WrapHandler(r.adminHandler.UpdateUserStatus))
		adminUsers.DELETE("/:id", shared_pres.WrapHandler(r.adminHandler.DeactivateUser))
		adminUsers.GET("/:id/sessions", shared_pres.WrapHandler(r.adminHandler.ListUserSessions))
		adminUsers.POST("/:id/sessions/revoke", shared_pres.WrapHandler(r.adminHandler.RevokeUserSessions))
	}
}
