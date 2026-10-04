package admin

import (
	"smart-wardrobe-be/services/account/contract/admin"
	"smart-wardrobe-be/services/account/internal/modules/admin/application/usecase"
	"smart-wardrobe-be/services/account/internal/modules/admin/infrastructure/persistence"
	"smart-wardrobe-be/services/account/internal/modules/admin/presentation/handler"
	"smart-wardrobe-be/services/account/internal/modules/admin/presentation/router"
	"smart-wardrobe-be/services/account/internal/modules/admin/presentation/worker"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	persistence.NewAdminAuditLogRepository,
	usecase.NewAuditLogUseCase,
	// 026 T066: the per-campaign audit READ, and the narrow contract the subscription
	// module consumes it through.
	//
	// `NewCampaignAuditLookupUseCase` is a SEPARATE use case rather than a method on
	// `AuditLogUseCase`, and the two providers are separate for the same reason: the
	// administration screen's collaborator and another module's narrow view answer
	// different questions, and putting the campaign's question on the screen's interface
	// would make every future caller of that screen depend on it.
	usecase.NewCampaignAuditLookupUseCase,
	// 026 T064: the adapter that hands `subscription` an `ICampaignAuditLookup`. It is
	// NOT `NewAuditLogContract` narrowed, because that contract is served over gRPC and
	// everything about its shape is a wire format (research.md R8).
	contract.NewCampaignAuditLookup,
	contract.NewAuditLogContract,
	handler.NewAdminAuditHandler,
	router.NewRouter,
	worker.NewAuditLogConsumerWorker,
	worker.NewAuditCleanupWorker,
)
