package shared

import (
	infra_media "smart-wardrobe-be/internal/shared/infrastructure/media"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	infra_media.NewCloudinaryService,
)
