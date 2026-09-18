package presentation

import (
	"net/http"
	"smart-wardrobe-be/shared/application/constants/apperror"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	// DefaultSSEChunkDeadline applies to local progress streams (analysis,
	// broadcasting) — how long the connection may stay silent per event.
	DefaultSSEChunkDeadline = 90 * time.Second
	// DefaultAIChatChunkDeadline applies to AI chat streams: LLM providers can
	// stall longer between tokens than local broadcasters (spec 019 FR-003).
	DefaultAIChatChunkDeadline = 180 * time.Second
)

// Deadlines are package-level so streaming handlers can extend without each
// carrying config; they are seeded from config in routes.NewEngine.
var (
	sseChunkDeadline    atomic.Value // time.Duration
	aiChatChunkDeadline atomic.Value // time.Duration
)

func init() {
	ResetSSEDeadlines()
}

// ConfigureSSEDeadlines seeds the per-chunk deadlines from config. Zero or
// negative values fall back to the defaults.
func ConfigureSSEDeadlines(chunk, aiChat time.Duration) {
	if chunk <= 0 {
		chunk = DefaultSSEChunkDeadline
	}
	if aiChat <= 0 {
		aiChat = DefaultAIChatChunkDeadline
	}
	sseChunkDeadline.Store(chunk)
	aiChatChunkDeadline.Store(aiChat)
}

// ResetSSEDeadlines restores the built-in defaults (used by tests).
func ResetSSEDeadlines() {
	sseChunkDeadline.Store(DefaultSSEChunkDeadline)
	aiChatChunkDeadline.Store(DefaultAIChatChunkDeadline)
}

func currentChunkDeadline() time.Duration {
	return sseChunkDeadline.Load().(time.Duration)
}

func currentAIChatChunkDeadline() time.Duration {
	return aiChatChunkDeadline.Load().(time.Duration)
}

// InitSSE configures the necessary headers for SSE and returns http.Flusher
func InitSSE(c *gin.Context) (http.Flusher, error) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	ExtendSSEWriteDeadline(c)

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		return nil, apperror.NewInternalError("Trình duyệt hoặc máy chủ không hỗ trợ kết nối thời gian thực.")
	}
	return flusher, nil
}

// ExtendSSEWriteDeadline pushes the connection write deadline further out so a
// long-running event stream is not cut by the server-wide WriteTimeout.
// Call it before writing each SSE event (or periodically between events).
func ExtendSSEWriteDeadline(c *gin.Context) {
	rc := http.NewResponseController(c.Writer)
	_ = rc.SetWriteDeadline(time.Now().Add(currentChunkDeadline()))
}

// ExtendAIChatSSEWriteDeadline is the looser variant for AI chat streams where
// LLM providers may stall longer between tokens than local broadcasters.
func ExtendAIChatSSEWriteDeadline(c *gin.Context) {
	rc := http.NewResponseController(c.Writer)
	_ = rc.SetWriteDeadline(time.Now().Add(currentAIChatChunkDeadline()))
}
