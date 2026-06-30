package usecase

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	branderrors "smart-wardrobe-be/internal/modules/brand/application/errors"
	"strings"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func generateClaimToken() (string, error) {
	buf := make([]byte, 9)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(buf)
	if len(token) > 12 {
		token = token[:12]
	}
	return token, nil
}

func (uc *BrandClaimUseCase) checkClaimRateLimit(ctx context.Context, userID uuid.UUID, claimToken string, clientIP string) error {
	if uc.redisClient == nil || uc.cfg == nil {
		return branderrors.ErrClaimRateLimitUnavailable()
	}
	tokenHash := sha256.Sum256([]byte(claimToken))
	limits := []struct {
		key   string
		limit int
	}{
		{key: "claim:ip:" + strings.TrimSpace(clientIP), limit: uc.cfg.ClaimRateLimit.IPLimit},
		{key: "claim:user:" + userID.String(), limit: uc.cfg.ClaimRateLimit.UserLimit},
		{key: "claim:token:" + hex.EncodeToString(tokenHash[:]), limit: uc.cfg.ClaimRateLimit.TokenLimit},
	}
	window := uc.cfg.ClaimRateLimit.WindowSeconds
	for _, item := range limits {
		allowed, err := uc.consumeClaimRateLimit(ctx, item.key, item.limit, window)
		if err != nil {
			return branderrors.ErrClaimRateLimitUnavailable()
		}
		if !allowed {
			return branderrors.ErrClaimRateLimited()
		}
	}
	return nil
}

func (uc *BrandClaimUseCase) consumeClaimRateLimit(ctx context.Context, key string, limit int, windowSeconds int) (bool, error) {
	script := redis.NewScript(`
local current = redis.call("INCR", KEYS[1])
if current == 1 then
  redis.call("EXPIRE", KEYS[1], ARGV[1])
end
if current > tonumber(ARGV[2]) then
  return 0
end
return 1
`)
	res, err := script.Run(ctx, uc.redisClient, []string{key}, windowSeconds, limit).Int()
	if err != nil {
		return false, err
	}
	return res == 1, nil
}
