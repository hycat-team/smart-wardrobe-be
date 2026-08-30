package infrastructure

import (
	"smart-wardrobe-be/internal/modules/brand/infrastructure/caching"
	"smart-wardrobe-be/internal/modules/brand/infrastructure/persistence"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	persistence.NewBrandRepository,
	persistence.NewBrandMemberRepository,
	persistence.NewBrandCustomerRepository,
	persistence.NewBrandItemRepository,
	persistence.NewDigitalSampleResponseRepository,
	persistence.NewBrandBenefitRepository,
	persistence.NewBenefitRedemptionRepository,
	persistence.NewBrandConversationRepository,
	persistence.NewBrandConversationMessageRepository,
	persistence.NewLoyaltyProgramRepository,
	persistence.NewLoyaltyTierRepository,
	persistence.NewLoyaltyAccountRepository,
	persistence.NewLoyaltyPointTransactionRepository,
	persistence.NewLoyaltyPointLotRepository,
	persistence.NewBrandCustomerClaimRepository,
	caching.NewRedisBrandMembershipCache,
	caching.NewRedisBrandFeatureAccessCache,
)
