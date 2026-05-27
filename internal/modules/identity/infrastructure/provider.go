package infrastructure

import (
	"smart-wardrobe-be/internal/modules/identity/infrastructure/caching"
	"smart-wardrobe-be/internal/modules/identity/infrastructure/communication"
	"smart-wardrobe-be/internal/modules/identity/infrastructure/persistence"
	"smart-wardrobe-be/internal/modules/identity/infrastructure/security"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	persistence.NewUserRepository,
	persistence.NewRefreshTokenRepository,
	caching.NewRedisOtpService,
	communication.NewGmailSmtpService,
	security.NewBcryptPasswordHasher,
	security.NewRedisTokenBlacklistService,
)
