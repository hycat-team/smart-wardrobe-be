package presentation

import (
	"net/http"
	"smart-wardrobe-be/internal/shared/application/constants/apperror"

	"github.com/gin-gonic/gin"
)

// InitSSE thiết lập các headers cần thiết cho SSE và trả về http.Flusher
func InitSSE(c *gin.Context) (http.Flusher, error) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		return nil, apperror.NewInternalError("Máy chủ không hỗ trợ SSE.")
	}
	return flusher, nil
}

