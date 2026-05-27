package presentation

import (
	"smart-wardrobe-be/internal/modules/identity/presentation/handler"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	handler.NewAuthHandler,
	handler.NewMeHandler,
)
