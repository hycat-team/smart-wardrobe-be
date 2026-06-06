package presentation

import (
	"net/http"
	"smart-wardrobe-be/internal/shared/application/constants/apperror"

	"github.com/gin-gonic/gin"
)

// InitSSE configures the necessary headers for SSE and returns http.Flusher
func InitSSE(c *gin.Context) (http.Flusher, error) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		return nil, apperror.NewInternalError("Trình duyệt hoặc máy chủ không hỗ trợ kết nối thời gian thực.")
	}
	return flusher, nil
}

