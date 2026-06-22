package subscription

import (
	"smart-wardrobe-be/internal/modules/subscription/application"
	"smart-wardrobe-be/internal/modules/subscription/infrastructure"
	"smart-wardrobe-be/internal/modules/subscription/presentation/handler"
	"smart-wardrobe-be/internal/modules/subscription/presentation/worker"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	application.ProviderSet,
	infrastructure.ProviderSet,
	handler.NewSubscriptionHandler,
	handler.NewBillingHandler,
	worker.NewSubscriptionRenewalWorker,
	worker.NewPaymentReconciliationWorker,
	worker.NewWebhookInboxWorker,
	worker.NewAIUsageReconciliationWorker,
)
