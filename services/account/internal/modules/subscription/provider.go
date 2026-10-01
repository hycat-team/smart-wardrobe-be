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
	// 024 T100: the bulk-grant limiter is a ROUTE dependency, so it is provided
	// here rather than inside the handler. Registering it in the set makes the
	// router's requirement visible to Wire instead of leaving a nil limiter to be
	// discovered on the first production request.
	shared_middleware.NewAdminGrantRateLimitMiddleware,
	// 024 T136: the narrow identity view the admin grant flow needs. Registered here
	// because the resolver's dependency changed type, and Wire should say so
	// instead of the resolver receiving a nil lookup at the first request.
	identity_contract.NewAdminGrantLookup,
	router.NewRouter,
	worker.NewSubscriptionRenewalWorker,
	worker.NewPaymentReconciliationWorker,
	worker.NewWebhookInboxWorker,
	// 025 T055: consumes `identity.user.created` and grants the campaign plan.
	// Registered here so the worker's dependency on the campaign use case is a
	// DI-time check rather than a nil pointer on the first account created after a
	// campaign opens.
	worker.NewCampaignGrantWorker,
)
