package application

import (
	"smart-wardrobe-be/internal/modules/brand/application/usecase/admin"
	"smart-wardrobe-be/internal/modules/brand/application/usecase/benefit"
	"smart-wardrobe-be/internal/modules/brand/application/usecase/brand"
	"smart-wardrobe-be/internal/modules/brand/application/usecase/chat"
	"smart-wardrobe-be/internal/modules/brand/application/usecase/claim"
	"smart-wardrobe-be/internal/modules/brand/application/usecase/item"
	"smart-wardrobe-be/internal/modules/brand/application/usecase/item/contractuc"
	"smart-wardrobe-be/internal/modules/brand/application/usecase/loyalty"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	admin.NewBrandAdminUseCase,
	benefit.NewBrandBenefitUseCase,
	brand.NewBrandUseCase,
	brand.NewBrandDashboardAnalyticsUseCase,
	chat.NewBrandChatUseCase,
	claim.NewBrandClaimUseCase,
	item.NewBrandItemUseCase,
	contractuc.NewBrandItemContractUseCase,
	item.NewBrandFashionEventUseCase,
	loyalty.NewBrandLoyaltyUseCase,
)
