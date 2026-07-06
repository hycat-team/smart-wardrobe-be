package application

import (
	"smart-wardrobe-be/internal/modules/brand/application/usecase/benefit"
	"smart-wardrobe-be/internal/modules/brand/application/usecase/brand"
	"smart-wardrobe-be/internal/modules/brand/application/usecase/chat"
	"smart-wardrobe-be/internal/modules/brand/application/usecase/claim"
	"smart-wardrobe-be/internal/modules/brand/application/usecase/item"
	"smart-wardrobe-be/internal/modules/brand/application/usecase/loyalty"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	benefit.NewBrandBenefitUseCase,
	brand.NewBrandUseCase,
	chat.NewBrandChatUseCase,
	claim.NewBrandClaimUseCase,
	item.NewBrandItemUseCase,
	loyalty.NewBrandLoyaltyUseCase,
)
