package router

import (
	"smart-wardrobe-be/internal/config"
	identity_contract "smart-wardrobe-be/internal/modules/identity/presentation/contract"
	subscription_handler "smart-wardrobe-be/internal/modules/subscription/presentation/handler"
	"smart-wardrobe-be/internal/shared/domain/constants/roleslug"
	shared_pres "smart-wardrobe-be/internal/shared/presentation"
	shared_middleware "smart-wardrobe-be/internal/shared/presentation/middleware"

	"github.com/gin-gonic/gin"
)

type Router struct {
	cfg                 *config.Config
	subscriptionHandler *subscription_handler.SubscriptionHandler
	billingHandler      *subscription_handler.BillingHandler
	auth                identity_contract.IAuthMiddleware
}

func NewRouter(
	cfg *config.Config,
	h *subscription_handler.SubscriptionHandler,
	b *subscription_handler.BillingHandler,
	auth identity_contract.IAuthMiddleware,
) *Router {
	return &Router{
		cfg:                 cfg,
		subscriptionHandler: h,
		billingHandler:      b,
		auth:                auth,
	}
}

func (r *Router) Init(group *gin.RouterGroup) {
	subApi := group.Group("/subscriptions")

	// Unauthenticated public payment endpoints
	subApi.POST("/payos-webhook", shared_pres.WrapHandler(r.billingHandler.ProcessPayOSWebhook))
	subApi.GET("/plans", shared_pres.WrapHandler(r.subscriptionHandler.GetPlans))

	// Authenticated subscription endpoints
	authSubApi := subApi.Group("")
	authSubApi.Use(r.auth.Handle(), shared_middleware.RolesAuthorize(roleslug.User))
	{
		authSubApi.GET("/me", shared_pres.WrapHandler(r.subscriptionHandler.GetUserSubscriptionOverview))
		authSubApi.GET("/me/daily-quota", shared_pres.WrapHandler(r.subscriptionHandler.GetDailyQuota))
		authSubApi.PUT("/me/auto-renew", shared_pres.WrapHandler(r.subscriptionHandler.SetAutoRenewStatus))
		authSubApi.GET("/me/wallet", shared_pres.WrapHandler(r.billingHandler.GetWallet))
		authSubApi.GET("/me/wallet/statements", shared_pres.WrapHandler(r.billingHandler.GetWalletStatements))

		paymentWriteApi := authSubApi.Group("")
		{
			paymentWriteApi.POST("/me/wallet/topup", shared_pres.WrapHandler(r.billingHandler.CreateWalletTopUp))
			paymentWriteApi.POST("/me/purchase", shared_pres.WrapHandler(r.billingHandler.CreateDirectPurchase))
			paymentWriteApi.POST("/me/purchase-with-wallet", shared_pres.WrapHandler(r.billingHandler.PurchasePlanWithWallet))
		}
	}
}
