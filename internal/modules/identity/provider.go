package identity

import (
	"smart-wardrobe-be/internal/modules/identity/application"
	"smart-wardrobe-be/internal/modules/identity/infrastructure"
	"smart-wardrobe-be/internal/modules/identity/presentation"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	presentation.ProviderSet,
	application.ProviderSet,
	infrastructure.ProviderSet,
)
