package sse

import (
	"sync"
)

// ISSEManager defines the interface for the Server-Sent Events manager.
type ISSEManager interface {
	Subscribe(taskId string) chan []byte
	Unsubscribe(taskId string, ch chan []byte)
	Publish(taskId string, data []byte)
}

// SSEManager implements ISSEManager to manage thread-safe event subscriptions.
type SSEManager struct {
	mu       sync.RWMutex
	channels map[string][]chan []byte // taskId -> list of subscriber channels
}

// NewSSEManager creates a new instance of SSEManager.
func NewSSEManager() *SSEManager {
	return &SSEManager{
		channels: make(map[string][]chan []byte),
	}
}

// Subscribe registers a new subscriber channel for a taskId.
func (m *SSEManager) Subscribe(taskId string) chan []byte {
	m.mu.Lock()
	defer m.mu.Unlock()

	ch := make(chan []byte, 10)
	m.channels[taskId] = append(m.channels[taskId], ch)
	return ch
}

// Unsubscribe closes and removes a subscriber channel for a taskId.
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
			break
		}
	}

	if len(m.channels[taskId]) == 0 {
		delete(m.channels, taskId)
	}
}

// Publish broadcasts data to all active subscriber channels for a taskId.
func (m *SSEManager) Publish(taskId string, data []byte) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	channels, exists := m.channels[taskId]
	if !exists {
		return
	}

	for _, ch := range channels {
		select {
		case ch <- data:
		default:
			// Avoid blocking if the receiver is slow or disconnected
		}
	}
}
