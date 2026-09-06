package presentation

import (
	"net/http"
	"smart-wardrobe-be/internal/shared/application/constants/apperror"
	"time"

	"github.com/gin-gonic/gin"
)

// SSEChunkWriteDeadline is how far the connection write deadline is pushed
// ahead on every streamed SSE event. It keeps long-running streams alive even
// when the server-wide WriteTimeout is shorter (spec 019 FR-003).
const SSEChunkWriteDeadline = 90 * time.Second

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
	_ = rc.SetWriteDeadline(time.Now().Add(SSEChunkWriteDeadline))
}
