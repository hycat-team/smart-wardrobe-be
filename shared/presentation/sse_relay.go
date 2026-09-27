package presentation

import (
	"context"
	"strings"

	"smart-wardrobe-be/shared/application/sse"
	"smart-wardrobe-be/shared/pkg/logger"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// SSERelay subscribes to the shared realtime channel and forwards each received
// event into the local SSEManager (fan-out + replay buffer). It runs in the HTTP
// process only (APP_MODE=http|all) and is best-effort: Redis errors are logged,
// never propagated (feature 023, FR-016/FR-025).
type SSERelay struct {
	client  *redis.Client
	manager sse.ISSEManager
	log     logger.Interface
	pubsub  *redis.PubSub
}

// NewSSERelay builds the Redis Pub/Sub → local SSEManager relay.
func NewSSERelay(client *redis.Client, manager sse.ISSEManager, log logger.Interface) *SSERelay {
	return &SSERelay{client: client, manager: manager, log: log}
}

// Start begins consuming the shared channel. It is a no-op when Redis is absent.
func (r *SSERelay) Start(ctx context.Context) {
	if r.client == nil {
		sse.IncRealtimeRelayFailed()
		return
	}
	r.pubsub = r.client.PSubscribe(ctx, sse.RealtimeChannelPattern)
	go r.consume(ctx)
	r.log.Info("sse realtime relay started")
}

func (r *SSERelay) consume(ctx context.Context) {
	ch := r.pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				// Channel closed unexpectedly (not via Stop/ctx): count it so
				// operations can detect a relay outage (FR-026).
				select {
				case <-ctx.Done():
				default:
					sse.IncRealtimeRelayFailed()
				}
				return
			}
			taskId := strings.TrimPrefix(msg.Channel, sse.RealtimeChannelPrefix)
			if taskId == "" {
				continue
			}
			sse.IncRealtimeRelayTotal()
			r.manager.Publish(taskId, []byte(msg.Payload))
		}
	}
}

// Stop closes the underlying subscription, ending consume.
func (r *SSERelay) Stop() {
	if r.pubsub != nil {
		if err := r.pubsub.Close(); err != nil {
			r.log.Warn("sse realtime relay close failed", zap.Error(err))
		}
	}
}
