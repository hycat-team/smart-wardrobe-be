package brand

import (
	"smart-wardrobe-be/internal/modules/brand/application/usecase"
	uc_interfaces "smart-wardrobe-be/internal/modules/brand/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/brand/contract"
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
	persistence.NewBrandBenefitRepository,
	persistence.NewBenefitRedemptionRepository,
	usecase.NewBrandCoreUseCase,
	wire.Bind(new(contract.IBrandContract), new(uc_interfaces.IBrandCoreUseCase)),
	handler.NewBrandHandler,
	worker.NewLoyaltyPointExpiryWorker,
)
