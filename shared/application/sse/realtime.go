package sse

import "context"

// IRealtimePublisher pushes an analysis event onto a cross-process channel so a
// relay running in the HTTP process can fan it out to connected SSE clients
// (feature 023, FR-016). Implementations MUST treat delivery as best-effort:
// transport failures are logged and swallowed, never propagated to the caller
// (FR-025).
type IRealtimePublisher interface {
	Publish(ctx context.Context, taskId string, data []byte) error
}

// IRealtimeRelay subscribes to the shared realtime channel and forwards
// received events into the local SSEManager (fan-out + replay buffer). It runs
// in the HTTP process only (APP_MODE=http|all).
type IRealtimeRelay interface {
	Start(ctx context.Context)
	Stop()
}

// RealtimeChannel returns the shared channel name for a session/task id.
func RealtimeChannel(taskId string) string {
	return RealtimeChannelPrefix + taskId
}

// RealtimeChannelPrefix is prepended to a task id to build its channel name.
const RealtimeChannelPrefix = "sse:task:"

// RealtimeChannelPattern is the PSubscribe glob covering every session channel.
const RealtimeChannelPattern = RealtimeChannelPrefix + "*"
