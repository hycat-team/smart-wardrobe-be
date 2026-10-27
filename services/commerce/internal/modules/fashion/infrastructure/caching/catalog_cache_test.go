package caching

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	shared_caching "smart-wardrobe-be/shared/infrastructure/caching"

	"github.com/google/uuid"
)

// 033 T049 - the catalog id cache (plan D5). These tests stay entirely in
// memory: the real Redis implementation is only a thin JSON wrapper, so an
// in-memory ICacheService proves miss/hit, key naming, TTL and fail-soft
// behaviour without a broker or network call.

// fakeCacheService is an in-memory ICacheService.
type fakeCacheService struct {
	values  map[string]string
	ttls    map[string]time.Duration
	setKeys []string
	getErr  error
	setErr  error
}

func newFakeCacheService() *fakeCacheService {
	return &fakeCacheService{
		values: map[string]string{},
		ttls:   map[string]time.Duration{},
	}
}

func (f *fakeCacheService) Get(_ context.Context, key string, dest interface{}) error {
	if f.getErr != nil {
		return f.getErr
	}
	raw, ok := f.values[key]
	if !ok {
		return shared_caching.ErrKeyNotFound
	}
	return json.Unmarshal([]byte(raw), dest)
}

func (f *fakeCacheService) Set(_ context.Context, key string, value interface{}, ttl time.Duration) error {
	if f.setErr != nil {
		return f.setErr
	}
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	f.values[key] = string(data)
	f.ttls[key] = ttl
	f.setKeys = append(f.setKeys, key)
	return nil
}

func (f *fakeCacheService) GetRaw(context.Context, string) (string, error) { return "", nil }
func (f *fakeCacheService) SetRaw(context.Context, string, string, time.Duration) error {
	return nil
}
func (f *fakeCacheService) Del(_ context.Context, keys ...string) error {
	for _, key := range keys {
		delete(f.values, key)
	}
	return nil
}
func (f *fakeCacheService) Exists(_ context.Context, keys ...string) (bool, error) {
	for _, key := range keys {
		if _, ok := f.values[key]; ok {
			return true, nil
		}
	}
	return false, nil
}
func (f *fakeCacheService) MGet(context.Context, interface{}, ...string) error { return nil }

func TestCatalogCacheMissThenHit(t *testing.T) {
	backend := newFakeCacheService()
	cache := NewRedisPersonalizationCatalogCache(backend)
	ctx := context.Background()

	ids, err := cache.Get(ctx)
	if err != nil {
		t.Fatalf("Get on empty cache: %v", err)
	}
	if len(ids) != 0 {
		t.Fatalf("Get on empty cache = %v, want a miss", ids)
	}

	wanted := []uuid.UUID{uuid.New(), uuid.New()}
	if err := cache.Set(ctx, wanted); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if len(backend.setKeys) != 1 || backend.setKeys[0] != personalizationCatalogCacheKey {
		t.Fatalf("cache key = %v, want [%s]", backend.setKeys, personalizationCatalogCacheKey)
	}
	if ttl := backend.ttls[personalizationCatalogCacheKey]; ttl != 5*time.Minute {
		t.Fatalf("ttl = %v, want 5m", ttl)
	}

	got, err := cache.Get(ctx)
	if err != nil {
		t.Fatalf("Get after Set: %v", err)
	}
	if len(got) != len(wanted) {
		t.Fatalf("Get after Set = %v, want %v", got, wanted)
	}
	for i := range wanted {
		if got[i] != wanted[i] {
			t.Fatalf("Get after Set = %v, want %v", got, wanted)
		}
	}
}

func TestCatalogCacheNilBackendIsAlwaysAMiss(t *testing.T) {
	cache := NewRedisPersonalizationCatalogCache(nil)
	ctx := context.Background()

	ids, err := cache.Get(ctx)
	if err != nil || len(ids) != 0 {
		t.Fatalf("Get on nil backend = (%v, %v), want (nil, nil)", ids, err)
	}
	if err := cache.Set(ctx, []uuid.UUID{uuid.New()}); err != nil {
		t.Fatalf("Set on nil backend must be a no-op, got %v", err)
	}
}

func TestCatalogCacheBackendErrorDegradesToMiss(t *testing.T) {
	backend := newFakeCacheService()
	backend.getErr = errors.New("redis unavailable")
	cache := NewRedisPersonalizationCatalogCache(backend)

	ids, err := cache.Get(context.Background())
	if err != nil {
		t.Fatalf("Get must swallow backend errors, got %v", err)
	}
	if len(ids) != 0 {
		t.Fatalf("Get on backend error = %v, want a miss", ids)
	}

	backend.setErr = errors.New("redis unavailable")
	if err := cache.Set(context.Background(), []uuid.UUID{uuid.New()}); err == nil {
		t.Fatal("Set must surface the write error so the caller can log it")
	}
}

func TestCatalogCacheDoesNotStoreEmptyList(t *testing.T) {
	backend := newFakeCacheService()
	cache := NewRedisPersonalizationCatalogCache(backend)

	if err := cache.Set(context.Background(), nil); err != nil {
		t.Fatalf("Set(nil): %v", err)
	}
	if len(backend.setKeys) != 0 {
		t.Fatalf("Set(nil) wrote keys %v, want no write (empty list means 'unknown')", backend.setKeys)
	}
}
