package brand

import (
	"smart-wardrobe-be/internal/modules/brand/application/usecase"
	"smart-wardrobe-be/internal/modules/brand/infrastructure/persistence"
	"smart-wardrobe-be/internal/modules/brand/presentation/handler"
	"smart-wardrobe-be/internal/modules/brand/presentation/worker"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	persistence.NewBrandRepository,
	persistence.NewBrandMemberRepository,
	persistence.NewBrandCustomerRepository,
	persistence.NewLoyaltyProgramRepository,
	persistence.NewLoyaltyTierRepository,
	persistence.NewLoyaltyAccountRepository,
	persistence.NewLoyaltyPointTransactionRepository,
	persistence.NewLoyaltyPointLotRepository,
	persistence.NewBrandCustomerClaimRepository,
	usecase.NewBrandCoreUseCase,
	handler.NewBrandHandler,
	worker.NewLoyaltyPointExpiryWorker,
)
