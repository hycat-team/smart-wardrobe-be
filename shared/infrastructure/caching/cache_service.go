package caching

import (
	"context"
	"time"
)

// ICacheService is an infrastructure-level abstraction for cache operations.
// Used only by other infrastructure implementations.
type ICacheService interface {
	// Get retrieves a value and unmarshals it into dest.
	Get(ctx context.Context, key string, dest interface{}) error
	// Set marshals value to JSON and stores it with TTL.
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	// GetRaw retrieves a raw string value (no unmarshaling).
	GetRaw(ctx context.Context, key string) (string, error)
	// SetRaw stores a raw string value with TTL (no marshaling).
	SetRaw(ctx context.Context, key string, value string, ttl time.Duration) error
	// Del removes one or more keys.
	Del(ctx context.Context, keys ...string) error
	// Exists checks if one or more keys exist.
	Exists(ctx context.Context, keys ...string) (bool, error)
	// MGet retrieves multiple values.
	MGet(ctx context.Context, dest interface{}, keys ...string) error
}
