package sse

import (
	"sync"
	"time"

	"smart-wardrobe-be/shared/config"
)

// ISSEManager defines the interface for the Server-Sent Events manager.
type ISSEManager interface {
	Subscribe(taskId string) chan []byte
	Unsubscribe(taskId string, ch chan []byte)
	Publish(taskId string, data []byte)
}

// subscriberBufferSize is the per-subscriber channel capacity for live events.
const subscriberBufferSize = 10

// taskBuffer keeps the most recent events of one analysis session so a client
// that reconnects can replay them (feature 023, FR-006).
type taskBuffer struct {
	events  [][]byte
	expires time.Time
}

// SSEManager implements ISSEManager to manage thread-safe event subscriptions
// plus a per-session replay buffer.
type SSEManager struct {
	mu       sync.RWMutex
	channels map[string][]chan []byte // taskId -> subscriber channels
	buffers  map[string]*taskBuffer   // taskId -> replay buffer
	maxSize  int
	ttl      time.Duration
	now      func() time.Time
}

// NewSSEManager creates a manager with the configured replay buffer size and
// TTL (defaults 100 events / 300s when config is missing or non-positive).
func NewSSEManager(cfg *config.Config) *SSEManager {
	maxSize := 100
	ttlSeconds := 300
	if cfg != nil {
		if cfg.Server.SSE.ReplayBufferSize > 0 {
			maxSize = cfg.Server.SSE.ReplayBufferSize
		}
		if cfg.Server.SSE.ReplayBufferTTLSeconds > 0 {
			ttlSeconds = cfg.Server.SSE.ReplayBufferTTLSeconds
		}
	}
	return &SSEManager{
		channels: make(map[string][]chan []byte),
		buffers:  make(map[string]*taskBuffer),
		maxSize:  maxSize,
		ttl:      time.Duration(ttlSeconds) * time.Second,
		now:      time.Now,
	}
}

// Subscribe registers a new subscriber channel for a taskId and immediately
// replays the buffered events (FIFO) so late reconnects do not miss updates.
func (m *SSEManager) Subscribe(taskId string) chan []byte {
	m.mu.Lock()
	defer m.mu.Unlock()

	ch := make(chan []byte, m.subscriberCap())
	if buf := m.activeBufferLocked(taskId); buf != nil {
		for _, ev := range buf.events {
			select {
			case ch <- ev:
			default:
			}
		}
	}
	m.channels[taskId] = append(m.channels[taskId], ch)
	IncRealtimeSubscribers()
	return ch
}

// Unsubscribe closes and removes a subscriber channel for a taskId. The replay
// buffer is intentionally kept so a later reconnect can still replay.
func (m *SSEManager) Unsubscribe(taskId string, ch chan []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()

	channels, exists := m.channels[taskId]
	if !exists {
		return
	}

	for i, c := range channels {
		if c == ch {
			close(c)
			m.channels[taskId] = append(channels[:i], channels[i+1:]...)
			DecRealtimeSubscribers()
			break
		}
	}

	if len(m.channels[taskId]) == 0 {
		delete(m.channels, taskId)
	}
}

// Publish buffers data for replay and broadcasts it to all active subscriber
// channels for a taskId.
func (m *SSEManager) Publish(taskId string, data []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()

	buf := m.activeBufferLocked(taskId)
	if buf == nil {
		buf = &taskBuffer{}
		m.buffers[taskId] = buf
	}
	buf.events = append(buf.events, data)
	if m.maxSize > 0 && len(buf.events) > m.maxSize {
		buf.events = buf.events[len(buf.events)-m.maxSize:]
	}
	buf.expires = m.now().Add(m.ttl)

	channels, exists := m.channels[taskId]
	if !exists {
		return
	}
	for _, ch := range channels {
		select {
		case ch <- data:
		default:
			// Avoid blocking if the receiver is slow or disconnected.
		}
	}
}

// activeBufferLocked returns the live buffer for taskId, lazily evicting it when
// its TTL has elapsed. Callers MUST hold m.mu.
func (m *SSEManager) activeBufferLocked(taskId string) *taskBuffer {
	buf, ok := m.buffers[taskId]
	if !ok {
		return nil
	}
	if m.now().After(buf.expires) {
		delete(m.buffers, taskId)
		return nil
	}
	return buf
}

// subscriberCap sizes a subscriber channel so a full replay fits without drops.
func (m *SSEManager) subscriberCap() int {
	if m.maxSize > subscriberBufferSize {
		return m.maxSize
	}
	return subscriberBufferSize
}
