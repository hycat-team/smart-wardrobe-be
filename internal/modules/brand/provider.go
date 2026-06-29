package brand

import (
	"smart-wardrobe-be/internal/modules/brand/application/usecase"
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
	persistence.NewBrandConversationRepository,
	persistence.NewBrandConversationMessageRepository,
	persistence.NewBrandItemRepository,
	persistence.NewDigitalSampleResponseRepository,
	usecase.NewBrandUseCase,
	usecase.NewBrandLoyaltyUseCase,
	usecase.NewBrandBenefitUseCase,
	usecase.NewBrandItemUseCase,
	usecase.NewBrandChatUseCase,
	usecase.NewBrandClaimUseCase,
	contract.NewBrandContract,
	handler.NewBrandPortalHandler,
	handler.NewBrandLoyaltyHandler,
	handler.NewBrandChatHandler,
	handler.NewBrandHandler,
	worker.NewLoyaltyPointExpiryWorker,
)
