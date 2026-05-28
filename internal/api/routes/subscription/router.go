package subscription

import (
	"smart-wardrobe-be/internal/api/middleware"
	subscription_handler "smart-wardrobe-be/internal/modules/subscription/presentation/handler"
	shared_pres "smart-wardrobe-be/internal/shared/presentation"

	"github.com/gin-gonic/gin"
)

type SubscriptionRouter struct {
	quotaHandler   *subscription_handler.DailyQuotaHandler
	billingHandler *subscription_handler.BillingHandler
	authMiddleware *middleware.AuthMiddleware
}

func NewRouter(
	h *subscription_handler.DailyQuotaHandler,
	b *subscription_handler.BillingHandler,
	m *middleware.AuthMiddleware,
) *SubscriptionRouter {
	return &SubscriptionRouter{
		quotaHandler:   h,
		billingHandler: b,
		authMiddleware: m,
	}
}

func (r *SubscriptionRouter) Init(group *gin.RouterGroup) {
	subApi := group.Group("/subscriptions")
	
	// Unauthenticated public payment endpoints
	subApi.POST("/payos-webhook", shared_pres.WrapHandler(r.billingHandler.ProcessPayOSWebhook))
	subApi.GET("/plans", shared_pres.WrapHandler(r.billingHandler.GetPlans))

	// Authenticated subscription endpoints
	authSubApi := subApi.Group("")
	authSubApi.Use(r.authMiddleware.Handle())
	{
		authSubApi.GET("/me/daily-quota", shared_pres.WrapHandler(r.quotaHandler.GetDailyQuota))
		authSubApi.PATCH("/me/toggle-auto-renew", shared_pres.WrapHandler(r.quotaHandler.ToggleAutoRenew))
		authSubApi.GET("/me/wallet", shared_pres.WrapHandler(r.billingHandler.GetWallet))
		authSubApi.GET("/me/wallet/statements", shared_pres.WrapHandler(r.billingHandler.GetWalletStatements))
		authSubApi.POST("/me/wallet/topup", shared_pres.WrapHandler(r.billingHandler.CreateWalletTopUp))
		authSubApi.POST("/me/purchase", shared_pres.WrapHandler(r.billingHandler.CreateDirectPurchase))
	}
}
