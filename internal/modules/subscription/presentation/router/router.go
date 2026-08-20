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
	cfg                      *config.Config
	subscriptionHandler      *subscription_handler.SubscriptionHandler
	billingHandler           *subscription_handler.BillingHandler
	adminBillingHandler      *subscription_handler.AdminBillingHandler
	adminSubscriptionHandler *subscription_handler.AdminSubscriptionHandler
	auth                     identity_contract.IAuthMiddleware
}

func NewRouter(
	cfg *config.Config,
	h *subscription_handler.SubscriptionHandler,
	b *subscription_handler.BillingHandler,
	ab *subscription_handler.AdminBillingHandler,
	as *subscription_handler.AdminSubscriptionHandler,
	auth identity_contract.IAuthMiddleware,
) *Router {
	return &Router{
		cfg:                      cfg,
		subscriptionHandler:      h,
		billingHandler:           b,
		adminBillingHandler:      ab,
		adminSubscriptionHandler: as,
		auth:                     auth,
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

	adminBilling := group.Group("/admin/subscriptions/transactions")
	adminBilling.Use(r.auth.Handle(), shared_middleware.RolesAuthorize(roleslug.Admin))
	{
		adminBilling.GET("", shared_pres.WrapHandler(r.adminBillingHandler.ListTransactions))
		adminBilling.GET("/:id", shared_pres.WrapHandler(r.adminBillingHandler.GetTransaction))
		adminBilling.POST("/:orderCode/retry-investigation", shared_pres.WrapHandler(r.adminBillingHandler.RetryInvestigation))
		adminBilling.POST("/:orderCode/resolve-paid", shared_pres.WrapHandler(r.adminBillingHandler.ResolveAsPaid))
		adminBilling.POST("/:orderCode/resolve-cancelled", shared_pres.WrapHandler(r.adminBillingHandler.ResolveAsCancelled))
		adminBilling.POST("/:orderCode/resolve-creation-failed", shared_pres.WrapHandler(r.adminBillingHandler.ResolveAsCreationFailed))
	}

	adminPlans := group.Group("/admin/subscriptions/plans")
	adminPlans.Use(r.auth.Handle(), shared_middleware.RolesAuthorize(roleslug.Admin))
	{
		adminPlans.GET("", shared_pres.WrapHandler(r.adminSubscriptionHandler.ListPlans))
		adminPlans.POST("", shared_pres.WrapHandler(r.adminSubscriptionHandler.CreatePlan))
		adminPlans.PUT("/:id", shared_pres.WrapHandler(r.adminSubscriptionHandler.UpdatePlan))
		adminPlans.PATCH("/:id/status", shared_pres.WrapHandler(r.adminSubscriptionHandler.SetPlanActive))
	}

	adminSubs := group.Group("/admin/subscriptions")
	adminSubs.Use(r.auth.Handle(), shared_middleware.RolesAuthorize(roleslug.Admin))
	{
		adminSubs.GET("", shared_pres.WrapHandler(r.adminSubscriptionHandler.ListSubscriptions))
		adminSubs.POST("/:userId/cancel", shared_pres.WrapHandler(r.adminSubscriptionHandler.CancelSubscription))
		adminSubs.POST("/:userId/grant", shared_pres.WrapHandler(r.adminSubscriptionHandler.GrantSubscription))
	}
}
