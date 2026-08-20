package admin

import (
	"smart-wardrobe-be/internal/modules/admin/application/usecase"
	"smart-wardrobe-be/internal/modules/admin/infrastructure/persistence"
	"smart-wardrobe-be/internal/modules/admin/presentation/contract"
	"smart-wardrobe-be/internal/modules/admin/presentation/handler"
	"smart-wardrobe-be/internal/modules/admin/presentation/router"
	"smart-wardrobe-be/internal/modules/admin/presentation/worker"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	persistence.NewAdminAuditLogRepository,
	usecase.NewAuditLogUseCase,
	contract.NewAuditLogContract,
	handler.NewAdminAuditHandler,
	router.NewRouter,
	worker.NewAuditLogConsumerWorker,
	worker.NewAuditCleanupWorker,
)
