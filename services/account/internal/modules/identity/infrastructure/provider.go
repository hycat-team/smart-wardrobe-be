package infrastructure

import (
	"smart-wardrobe-be/services/account/internal/modules/identity/infrastructure/caching"
	"smart-wardrobe-be/services/account/internal/modules/identity/infrastructure/persistence"
	"smart-wardrobe-be/services/account/internal/modules/identity/infrastructure/security"
	sharedsecurity "smart-wardrobe-be/shared/infrastructure/security"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	persistence.NewUserRepository,
	persistence.NewRefreshTokenRepository,
	caching.NewRedisOtpService,
	security.NewBcryptPasswordHasher,
	sharedsecurity.NewRedisTokenBlacklistService,
)
