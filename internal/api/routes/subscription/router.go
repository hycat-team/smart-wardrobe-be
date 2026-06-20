package subscription

import (
	"net/http"
	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/api/middleware"
	subscription_handler "smart-wardrobe-be/internal/modules/subscription/presentation/handler"
	"smart-wardrobe-be/internal/shared/application/constants/apperror"
	"smart-wardrobe-be/internal/shared/domain/constants/roleslug"
	shared_pres "smart-wardrobe-be/internal/shared/presentation"

	"github.com/gin-gonic/gin"
)

type SubscriptionRouter struct {
	cfg                 *config.Config
	subscriptionHandler *subscription_handler.SubscriptionHandler
	billingHandler      *subscription_handler.BillingHandler
	authMiddleware      *middleware.AuthMiddleware
}

func NewRouter(
	cfg *config.Config,
	h *subscription_handler.SubscriptionHandler,
	b *subscription_handler.BillingHandler,
	m *middleware.AuthMiddleware,
) *SubscriptionRouter {
	return &SubscriptionRouter{
		cfg:                 cfg,
		subscriptionHandler: h,
		billingHandler:      b,
		authMiddleware:      m,
	}
}

func (r *SubscriptionRouter) Init(group *gin.RouterGroup) {
	subApi := group.Group("/subscriptions")

	// Unauthenticated public payment endpoints
	subApi.POST("/payos-webhook", shared_pres.WrapHandler(r.billingHandler.ProcessPayOSWebhook))
	subApi.GET("/plans", shared_pres.WrapHandler(r.subscriptionHandler.GetPlans))

	// Authenticated subscription endpoints
	authSubApi := subApi.Group("")
	authSubApi.Use(r.authMiddleware.Handle(), middleware.RolesAuthorize(roleslug.User))
	{
		authSubApi.GET("/me", shared_pres.WrapHandler(r.subscriptionHandler.GetUserSubscriptionOverview))
		authSubApi.GET("/me/daily-quota", shared_pres.WrapHandler(r.subscriptionHandler.GetDailyQuota))
		authSubApi.PUT("/me/auto-renew", shared_pres.WrapHandler(r.subscriptionHandler.SetAutoRenewStatus))
		authSubApi.GET("/me/wallet", shared_pres.WrapHandler(r.billingHandler.GetWallet))
		authSubApi.GET("/me/wallet/statements", shared_pres.WrapHandler(r.billingHandler.GetWalletStatements))

		// paymentWriteApi routes are disabled in production environment
		paymentWriteApi := authSubApi.Group("")
		if r.cfg.Server.Env == "production" {
			paymentWriteApi.Use(func(c *gin.Context) {
				c.JSON(http.StatusForbidden, apperror.NewError(
					http.StatusForbidden,
					"Tính năng tạm đóng",
					"Tính năng nạp tiền và đăng ký gói hội viên đang tạm đóng ở phiên bản thử nghiệm này.",
				))
				c.Abort()
			})
		}
		{
			paymentWriteApi.POST("/me/wallet/topup", shared_pres.WrapHandler(r.billingHandler.CreateWalletTopUp))
			paymentWriteApi.POST("/me/purchase", shared_pres.WrapHandler(r.billingHandler.CreateDirectPurchase))
			paymentWriteApi.POST("/me/purchase-with-wallet", shared_pres.WrapHandler(r.billingHandler.PurchasePlanWithWallet))
		}
	}
}
