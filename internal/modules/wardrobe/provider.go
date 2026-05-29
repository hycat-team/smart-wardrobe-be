package wardrobe

import (
	"smart-wardrobe-be/internal/modules/wardrobe/application/usecase"
	"smart-wardrobe-be/internal/modules/wardrobe/infrastructure/persistence"
	"smart-wardrobe-be/internal/modules/wardrobe/presentation/handler"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	persistence.NewWardrobeItemRepository,
	persistence.NewCategoryRepository,
	usecase.NewWardrobeUseCase,
	handler.NewWardrobeHandler,
)
