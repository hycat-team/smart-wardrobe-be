package subscription

import (
	"smart-wardrobe-be/services/account/internal/modules/subscription/application"
	"smart-wardrobe-be/services/account/internal/modules/subscription/infrastructure"
	"smart-wardrobe-be/services/account/internal/modules/subscription/presentation/handler"
	"smart-wardrobe-be/services/account/internal/modules/subscription/presentation/router"
	"smart-wardrobe-be/services/account/internal/modules/subscription/presentation/worker"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	application.ProviderSet,
	infrastructure.ProviderSet,
	handler.NewSubscriptionHandler,
	handler.NewBillingHandler,
	handler.NewAdminBillingHandler,
	handler.NewAdminSubscriptionHandler,
	router.NewRouter,
	worker.NewSubscriptionRenewalWorker,
	worker.NewPaymentReconciliationWorker,
	worker.NewWebhookInboxWorker,
)
