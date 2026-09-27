package messaging

import (
	"context"

	"smart-wardrobe-be/shared/application/sse"
	"smart-wardrobe-be/shared/pkg/logger"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// RedisRealtimePublisher publishes SSE events to a Redis Pub/Sub channel so the
// relay running in the HTTP process can deliver them to clients in both
// topologies (VPS APP_MODE=all and split http/worker on AWS). Publishing is
// best-effort: a Redis failure is logged and swallowed so it never blocks or
// fails the business flow (feature 023, FR-025).
type RedisRealtimePublisher struct {
	client *redis.Client
	log    logger.Interface
}

// NewRedisRealtimePublisher builds the Redis-backed realtime publisher.
func NewRedisRealtimePublisher(client *redis.Client, log logger.Interface) *RedisRealtimePublisher {
	return &RedisRealtimePublisher{client: client, log: log}
}

// Publish sends the event bytes to the task's channel. It always returns nil to
// keep the caller's business flow uninterrupted; transport errors are logged.
func (p *RedisRealtimePublisher) Publish(ctx context.Context, taskId string, data []byte) error {
	if p.client == nil {
		return nil
	}
	sse.IncRealtimePublishTotal()
	if err := p.client.Publish(ctx, sse.RealtimeChannel(taskId), data).Err(); err != nil {
		sse.IncRealtimePublishFailed()
		p.log.Warn("realtime publish failed",
			zap.String("task_id", taskId),
			zap.Error(err),
		)
	}
	return nil
}
