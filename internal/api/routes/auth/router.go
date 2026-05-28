package auth

import (
	"smart-wardrobe-be/internal/api/middleware"
	identity_handler "smart-wardrobe-be/internal/modules/identity/presentation/handler"
	shared_pres "smart-wardrobe-be/internal/shared/presentation"

	"github.com/gin-gonic/gin"
)

type AuthRouter struct {
	authHandler    *identity_handler.AuthHandler
	authMiddleware *middleware.AuthMiddleware
}

func NewRouter(h *identity_handler.AuthHandler, m *middleware.AuthMiddleware) *AuthRouter {
	return &AuthRouter{
		authHandler:    h,
		authMiddleware: m,
	}
}

func (r *AuthRouter) Init(group *gin.RouterGroup) {
	authApi := group.Group("/auth")
	{
		authApi.POST("/register", shared_pres.WrapHandler(r.authHandler.Register))
		authApi.POST("/register/confirm-otp", shared_pres.WrapHandler(r.authHandler.ConfirmRegisterOtp))
		authApi.POST("/login", shared_pres.WrapHandler(r.authHandler.Login))
		authApi.POST("/refresh-token", shared_pres.WrapHandler(r.authHandler.RefreshToken))
		authApi.POST("/forgot-password", shared_pres.WrapHandler(r.authHandler.ForgotPassword))
		authApi.POST("/forgot-password/confirm-otp", shared_pres.WrapHandler(r.authHandler.ConfirmForgotPasswordOtp))
		authApi.POST("/reset-password", shared_pres.WrapHandler(r.authHandler.ResetPassword))
	}

	privateAuth := authApi.Group("")
	privateAuth.Use(r.authMiddleware.Handle())
	{
		privateAuth.POST("/logout", shared_pres.WrapHandler(r.authHandler.Logout))
	}
}
