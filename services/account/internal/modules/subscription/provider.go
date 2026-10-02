package subscription

import (
	identity_contract "smart-wardrobe-be/services/account/contract/identity"
	"smart-wardrobe-be/services/account/internal/modules/subscription/application"
	"smart-wardrobe-be/services/account/internal/modules/subscription/infrastructure"
	"smart-wardrobe-be/services/account/internal/modules/subscription/presentation/handler"
	"smart-wardrobe-be/services/account/internal/modules/subscription/presentation/router"
	"smart-wardrobe-be/services/account/internal/modules/subscription/presentation/worker"
	shared_middleware "smart-wardrobe-be/shared/presentation/middleware"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	application.ProviderSet,
	infrastructure.ProviderSet,
	handler.NewSubscriptionHandler,
	handler.NewBillingHandler,
	handler.NewAdminBillingHandler,
	handler.NewAdminSubscriptionHandler,
	// 025 T081: the read-only operator look-up. It depends on
	// `ICampaignQueryUseCase` and on nothing else, so registering it here is what
	// makes FR-069 structural: the handler's only collaborator is an interface with
	// no write method anywhere behind it.
	handler.NewCampaignAdminHandler,
	// 024 T100: the bulk-grant limiter is a ROUTE dependency, so it is provided
	// here rather than inside the handler. Registering it in the set makes the
	// router's requirement visible to Wire instead of leaving a nil limiter to be
	// discovered on the first production request.
	shared_middleware.NewAdminGrantRateLimitMiddleware,
	// 024 T136: the narrow identity view the admin grant flow needs. Registered here
	// because the resolver's dependency changed type, and Wire should say so
	// instead of the resolver receiving a nil lookup at the first request.
	identity_contract.NewAdminGrantLookup,
	// 025 T099: the narrow identity view the catch-up sweep pages through. It is
	// a SEPARATE provider from `NewAdminGrantLookup` because the two narrow views
	// answer different questions, and widening either one to cover the other would
	// put a campaign-specific filter into the administrator grant flow.
	identity_contract.NewCampaignAccountLister,
	router.NewRouter,
	worker.NewSubscriptionRenewalWorker,
	worker.NewPaymentReconciliationWorker,
	worker.NewWebhookInboxWorker,
	// 025 T055: consumes `identity.user.created` and grants the campaign plan.
	// Registered here so the worker's dependency on the campaign use case is a
	// DI-time check rather than a nil pointer on the first account created after a
	// campaign opens.
	worker.NewCampaignGrantWorker,
	// 025 T099: the periodic catch-up sweep (FR-079, FR-083). Registered next to
	// the grant worker because the two are one mechanism: the queue is the fast
	// path and this is the one that guarantees the promise.
	worker.NewCampaignSweepWorker,
)
