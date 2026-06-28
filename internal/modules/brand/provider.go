package brand

import (
	"smart-wardrobe-be/internal/modules/brand/application/usecase"
	"smart-wardrobe-be/internal/modules/brand/infrastructure/persistence"
	"smart-wardrobe-be/internal/modules/brand/presentation/handler"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	persistence.NewBrandRepository,
	persistence.NewBrandMemberRepository,
	persistence.NewBrandCustomerRepository,
	usecase.NewBrandCoreUseCase,
	handler.NewBrandHandler,
)
