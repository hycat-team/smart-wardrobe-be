package router

import (
	identity_contract "smart-wardrobe-be/services/account/contract/identity"
	subscription_handler "smart-wardrobe-be/services/account/internal/modules/subscription/presentation/handler"
	"smart-wardrobe-be/shared/config"
	"smart-wardrobe-be/shared/domain/constants/roleslug"
	shared_pres "smart-wardrobe-be/shared/presentation"
	shared_middleware "smart-wardrobe-be/shared/presentation/middleware"

	"github.com/gin-gonic/gin"
)

type Router struct {
	cfg                      *config.Config
	subscriptionHandler      *subscription_handler.SubscriptionHandler
	billingHandler           *subscription_handler.BillingHandler
	adminBillingHandler      *subscription_handler.AdminBillingHandler
	adminSubscriptionHandler *subscription_handler.AdminSubscriptionHandler
	// 025 T081: the read-only operator look-up. It is a separate handler rather
	// than another method on `AdminSubscriptionHandler` because the look-up and
	// the grant are opposites in every way that matters: one reads and one writes,
	// one is reachable from a support page and one from a batch button, and one
	// must be provably incapable of spending anything (FR-069). A separate type
	// makes that provable from its constructor's parameter list.
	campaignAdminHandler *subscription_handler.CampaignAdminHandler
	auth                 identity_contract.IAuthMiddleware
	// 024 T100: the bulk-grant limiter. It lives on the route rather than in the
	// handler so it also covers requests rejected before the handler runs, and so
	// the two-tier rule reads in one place next to the route it protects.
	adminGrantRateLimiter *shared_middleware.AdminGrantRateLimitMiddleware
}

func NewRouter(
	cfg *config.Config,
	h *subscription_handler.SubscriptionHandler,
	b *subscription_handler.BillingHandler,
	ab *subscription_handler.AdminBillingHandler,
	as *subscription_handler.AdminSubscriptionHandler,
	cah *subscription_handler.CampaignAdminHandler,
	auth identity_contract.IAuthMiddleware,
	adminGrantRateLimiter *shared_middleware.AdminGrantRateLimitMiddleware,
) *Router {
	return &Router{
		cfg:                      cfg,
		subscriptionHandler:      h,
		billingHandler:           b,
		adminBillingHandler:      ab,
		adminSubscriptionHandler: as,
		campaignAdminHandler:     cah,
		auth:                     auth,
		adminGrantRateLimiter:    adminGrantRateLimiter,
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
		// Cancel stays. It is not a grant, and removing a working revocation
		// would leave an administrator no way to stop a subscription that was
		// granted in error.
		adminSubs.POST("/:userId/cancel", shared_pres.WrapHandler(r.adminSubscriptionHandler.CancelSubscription))
		// 024 T089/FR-060: POST /:userId/grant is GONE. It is replaced by the
		// batch endpoint below, which takes a list - a single account is a list
		// of one.
		//
		// ⚠️ THE NEW PATH CANNOT BE /admin/subscriptions/grants. `adminSubs`
		// already holds `POST /:userId/cancel`, and Gin refuses to register a
		// static segment ("grants") beside a parameter at the same level: the
		// conflict panics at STARTUP, so the whole account service would refuse
		// to boot. Hence a separate group. See research.md R5.
	}

	// 024 T089/FR-060: the one and only grant interface. Both limit tiers are
	// counted per administrator (024 T099), and the limiter runs AFTER auth
	// because the key IS the administrator's identity.
	adminGrants := group.Group("/admin/subscription-grants")
	adminGrants.Use(r.auth.Handle(), shared_middleware.RolesAuthorize(roleslug.Admin))
	{
		adminGrants.POST("", r.adminGrantRateLimit(), shared_pres.WrapHandler(r.adminSubscriptionHandler.GrantSubscriptionsBatch))
		// 024 T110: the repair tool. It shares the administrator group, and
		// therefore the same RolesAuthorize check (FR-089), but NOT the grant rate
		// limiter: a restore is a single deliberate operation an operator runs
		// once, not a request an administrator can repeat by accident.
		adminGrants.POST("/restore", shared_pres.WrapHandler(r.adminSubscriptionHandler.RestorePolicyGrants))
	}

	// 025 T082: the campaign look-up — a SEPARATE GROUP, and that is not a style
	// choice.
	//
	// ⚠️ IT MUST NOT BE NESTED UNDER `/admin/subscriptions`. That group already
	// holds `POST /:userId/cancel`, and Gin refuses to register a STATIC segment
	// ("claims", "eligibility") beside a PARAMETER at the same level. The conflict
	// PANICS AT STARTUP, so the whole account service would refuse to boot — a
	// read-only reporting feature taking down authentication, plans and payments.
	// This exact trap was already hit and written up at 024 R5; the comment on
	// `adminSubs` below records it the second time.
	adminCampaigns := group.Group("/admin/campaigns")
	adminCampaigns.Use(r.auth.Handle(), shared_middleware.RolesAuthorize(roleslug.Admin))
	{
		// FR-070 is answered by the group's own `RolesAuthorize(roleslug.Admin)`:
		// a normal account is refused with the standard 403 even when it knows the
		// path. There is deliberately no role check of our own — a second one
		// would be a second answer to the same question.
		adminCampaigns.GET("", r.adminGrantRateLimit(), shared_pres.WrapHandler(r.campaignAdminHandler.ListCampaigns))
		adminCampaigns.GET("/:code", r.adminGrantRateLimit(), shared_pres.WrapHandler(r.campaignAdminHandler.GetCampaign))
		// FR-071: the SAME limiter as the bulk grant, not a new one. It is a
		// per-administrator limit, which is what FR-071 asks for ("a round of
		// look-ups must not exhaust the system"), and reusing it means an operator
		// hammering this page spends the same budget as one hammering grants —
		// which is the honest accounting, because both are administrator actions
		// against the same deployment. Adding a second configuration for the same
		// limit would be a second answer waiting to be given (R10).
		adminCampaigns.GET("/:code/claims", r.adminGrantRateLimit(), shared_pres.WrapHandler(r.campaignAdminHandler.ListClaims))
		adminCampaigns.GET("/:code/eligibility/:userId", r.adminGrantRateLimit(), shared_pres.WrapHandler(r.campaignAdminHandler.GetAccountEligibility))
	}
}

// adminGrantRateLimit applies the two-tier administrator rule.
//
// The nil check is a wiring guard, not a feature: a nil limiter would panic on
// the first request instead of failing the build. It cannot happen through Wire
// — the constructor is a required parameter — so this exists only for tests and
// for anyone assembling the router by hand.
func (r *Router) adminGrantRateLimit() gin.HandlerFunc {
	if r.adminGrantRateLimiter == nil {
		return func(c *gin.Context) { c.Next() }
	}
	return r.adminGrantRateLimiter.Handle()
}
