package subscription

import (
	"smart-wardrobe-be/internal/modules/subscription/application"
	"smart-wardrobe-be/internal/modules/subscription/infrastructure"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	application.ProviderSet,
	infrastructure.ProviderSet,
)
