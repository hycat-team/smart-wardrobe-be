package subscription

import (
	"smart-wardrobe-be/internal/api/middleware"
	subscription_handler "smart-wardrobe-be/internal/modules/subscription/presentation/handler"
	shared_pres "smart-wardrobe-be/internal/shared/presentation"

	"github.com/gin-gonic/gin"
)

type SubscriptionRouter struct {
	quotaHandler   *subscription_handler.DailyQuotaHandler
	authMiddleware *middleware.AuthMiddleware
}

func NewRouter(h *subscription_handler.DailyQuotaHandler, m *middleware.AuthMiddleware) *SubscriptionRouter {
	return &SubscriptionRouter{
		quotaHandler:   h,
		authMiddleware: m,
	}
}

func (r *SubscriptionRouter) Init(group *gin.RouterGroup) {
	subApi := group.Group("/subscriptions")
	subApi.Use(r.authMiddleware.Handle())
	{
		subApi.GET("/daily-quota", shared_pres.WrapHandler(r.quotaHandler.GetDailyQuota))
	}
}
