package me

import (
	"smart-wardrobe-be/internal/api/middleware"
	identity_handler "smart-wardrobe-be/internal/modules/identity/presentation/handler"
	shared_pres "smart-wardrobe-be/internal/shared/presentation"

	"github.com/gin-gonic/gin"
)

type MeRouter struct {
	meHandler      *identity_handler.MeHandler
	authMiddleware *middleware.AuthMiddleware
}

func NewRouter(h *identity_handler.MeHandler, m *middleware.AuthMiddleware) *MeRouter {
	return &MeRouter{
		meHandler:      h,
		authMiddleware: m,
	}
}

func (r *MeRouter) Init(group *gin.RouterGroup) {
	meApi := group.Group("/me")
	meApi.Use(r.authMiddleware.Handle())
	{
		meApi.GET("", shared_pres.WrapHandler(r.meHandler.GetCurrentUser))
		meApi.PUT("", shared_pres.WrapHandler(r.meHandler.UpdateCurrentUser))
		meApi.PUT("/change-password", shared_pres.WrapHandler(r.meHandler.ChangePassword))
		meApi.GET("/avatar-signature", shared_pres.WrapHandler(r.meHandler.GetAvatarSignature))
		meApi.PUT("/avatar", shared_pres.WrapHandler(r.meHandler.UpdateAvatar))
	}
}
