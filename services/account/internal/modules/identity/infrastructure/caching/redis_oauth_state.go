package caching

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	identityiface "smart-wardrobe-be/services/account/internal/modules/identity/application/interface/identity"
	"smart-wardrobe-be/services/account/internal/modules/identity/application/vo"
	shared_infra_caching "smart-wardrobe-be/shared/infrastructure/caching"
)

const oauthStatePrefix = "oauth:google:state:"

// RedisOAuthStateStore stores one-time OAuth state for the web redirect flow.
type RedisOAuthStateStore struct {
	cache shared_infra_caching.ICacheService
}

func NewRedisOAuthStateStore(cache shared_infra_caching.ICacheService) identityiface.IOAuthStateStore {
	return &RedisOAuthStateStore{cache: cache}
}

func (s *RedisOAuthStateStore) Save(ctx context.Context, state string, data vo.OAuthState, ttl time.Duration) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return s.cache.SetRaw(ctx, oauthStatePrefix+state, string(payload), ttl)
}

func (s *RedisOAuthStateStore) Consume(ctx context.Context, state string) (*vo.OAuthState, error) {
	raw, err := s.cache.GetRaw(ctx, oauthStatePrefix+state)
	if err != nil {
		if errors.Is(err, shared_infra_caching.ErrKeyNotFound) {
			return nil, nil
		}
		return nil, err
	}

	_ = s.cache.Del(ctx, oauthStatePrefix+state)

	var data vo.OAuthState
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return nil, err
	}
	return &data, nil
}
