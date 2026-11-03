package caching

import (
	"context"
	"time"

	shared_infra_caching "smart-wardrobe-be/shared/infrastructure/caching"
)

const deliveredEmailPrefix = "email:delivered:"

// RedisDeliveryDeduplicator remembers delivered outbox rows for a retention
// window, so an at-least-once redelivery does not become a duplicate email.
//
// Redis rather than a table: the record only has to outlive the broker's
// redelivery window, losing it costs at most one duplicate email, and the email
// path must not take a database write per message. A Redis outage fails open —
// the worker logs and sends — because blocking the customer's receipt to protect
// against a duplicate is the worse trade.
type RedisDeliveryDeduplicator struct {
	cache shared_infra_caching.ICacheService
}

func NewRedisDeliveryDeduplicator(cache shared_infra_caching.ICacheService) *RedisDeliveryDeduplicator {
	return &RedisDeliveryDeduplicator{cache: cache}
}

func (d *RedisDeliveryDeduplicator) Seen(ctx context.Context, key string) (bool, error) {
	return d.cache.Exists(ctx, deliveredEmailPrefix+key)
}

func (d *RedisDeliveryDeduplicator) Mark(ctx context.Context, key string, ttl time.Duration) error {
	return d.cache.SetRaw(ctx, deliveredEmailPrefix+key, "1", ttl)
}
