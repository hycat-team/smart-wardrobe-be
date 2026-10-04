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
	// 026 T035: the campaign WRITE face. A separate handler, and the reason is
	// structural rather than stylistic: `CampaignAdminHandler`'s only collaborator is
	// an interface with no write method anywhere behind it, which is how FR-069's
	// read-only guarantee is kept (see the comment on that field). Adding a unit of
	// work and a registry writer to the same type would delete that guarantee in one
	// edit, so the two write to the same URL prefix from two different objects.
	campaignRegistryHandler *subscription_handler.CampaignRegistryHandler
	auth                    identity_contract.IAuthMiddleware
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
	crh *subscription_handler.CampaignRegistryHandler,
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
		campaignRegistryHandler:  crh,
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

		// 026 T035 (FR-052/FR-053): the create endpoint — the moment 026 exists for.
		//
		// IT JOINS THIS GROUP RATHER THAN GETTING A NEW ONE, and that is safe for a
		// reason the group comment above makes concrete: the Gin trap is a STATIC
		// segment registered BESIDE a PARAMETER at the same level. `POST ""` is the
		// same static path `GET ""` already uses, and no parameter segment was added at
		// this level — so there is no new conflict to panic at startup. The three
		// campaign write routes, by contrast, DO add parameters and are therefore
		// added to this same group in the order US2/US3 bring them, never nested under
		// `/admin/subscriptions` where `POST /:userId/cancel` lives.
		//
		// THE SAME `RolesAuthorize` AND THE SAME LIMITER, both inherited from the group
		// and the two lines below (FR-053). A second limiter for writes would make "an
		// administrator's budget" two numbers, and the first one to be spent would be
		// whichever the client happened to call.
		adminCampaigns.POST("", r.adminGrantRateLimit(), shared_pres.WrapHandler(r.campaignRegistryHandler.Create))
		// 026 T043 (FR-030/FR-031/FR-034): the force-close endpoint.
		//
		// IT JOINS THIS GROUP RATHER THAN GETTING A NEW ONE, for the reason the
		// comment above spells out: the Gin trap is a STATIC segment registered BESIDE
		// a parameter at the same level, and this route adds a `:code` parameter
		// under `POST`. The sibling `POST ""` and `POST "/restore"` in the
		// `adminGrants` group above are the precedent that this shape is safe.
		//
		// THE SAME `RolesAuthorize` AND THE SAME LIMITER, both inherited from the group
		// (FR-053). A close is the most consequential write the campaign registry has
		// — it is one-way and cannot be undone — so charging it the same
		// administrator budget as the other writes is what stops a retry loop from
		// spending it twice.
		adminCampaigns.POST("/:code/close", r.adminGrantRateLimit(), shared_pres.WrapHandler(r.campaignRegistryHandler.Close))
		// 026 T053 (FR-020..FR-024, FR-055): the edit endpoint.
		//
		// `PATCH "/:code"` REUSES THE SAME PARAMETER SEGMENT the read routes use, and
		// that is deliberate rather than incidental: a second parameter name for the
		// same value would be a second binding answer to "which campaign", and the
		// handler binds ONE `campaignPathParams` for all of them (contract §4).
		//
		// REGISTERED BESIDE `POST "/:code/close"` AND NOT BESIDE A STATIC SEGMENT, which
		// is the whole R11 trap: Gin panics at startup when a static path segment is
		// registered beside a parameter at the same level. `PATCH "/:code"` adds no
		// static segment, so the group stays valid — and
		// `campaign_route_registration_test.go` calls the real `Init` so a regression is
		// a `go test` failure instead of a service that refuses to boot.
		//
		// THE SAME `RolesAuthorize` AND THE SAME LIMITER, inherited from the group
		// (FR-053). An edit moves the budget a campaign is still allowed to hand out,
		// which makes it as consequential as the close.
		adminCampaigns.PATCH("/:code", r.adminGrantRateLimit(), shared_pres.WrapHandler(r.campaignRegistryHandler.Update))
		// 026 T067 (FR-053, FR-063): the audit trail of one campaign.
		//
		// IT JOINS THIS GROUP AND NOT A NEW ONE, for the reason the group comment makes
		// concrete: the Gin trap is a STATIC segment registered BESIDE a parameter at the
		// same level. `audit` is a static segment, but it sits at the level BELOW `:code`
		// — the same shape as the `claims` and `eligibility` routes above — so there is
		// no conflict to panic at startup, and
		// `campaign_no_delete_route_test.go` calls the real `Init` so a regression is a
		// `go test` failure rather than a service that refuses to boot.
		//
		// IT IS A `GET` ON THE READ-ONLY HANDLER even though the trail records writes.
		// Reading an audit trail changes nothing: the rows were written asynchronously by
		// the existing consumer, the grant path does not consult them, and nothing on
		// this route can spend a slot or grant a plan (FR-069). FR-053's role check and
		// limiter are inherited from the group like every other route above.
		adminCampaigns.GET("/:code/audit", r.adminGrantRateLimit(), shared_pres.WrapHandler(r.campaignAdminHandler.GetCampaignAudit))
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
