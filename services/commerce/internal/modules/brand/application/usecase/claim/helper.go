package claim

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	branderrors "smart-wardrobe-be/services/commerce/internal/modules/brand/application/errors"
	"strings"

	"github.com/google/uuid"
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
	if uc.rateLimiter == nil || uc.cfg == nil {
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
		allowed, err := uc.rateLimiter.Allow(ctx, item.key, item.limit, window)
		if err != nil {
			return branderrors.ErrClaimRateLimitUnavailable()
		}
		if !allowed {
			return branderrors.ErrClaimRateLimited()
		}
	}
	return nil
}
