package middleware

import (
	"context"
	"net/http"
	"time"

	"smart-wardrobe-be/internal/shared/application/constants/apperror"

	"github.com/gin-gonic/gin"
)

func GlobalTimeoutMiddleware(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)
		c.Next()

		if ctx.Err() == context.DeadlineExceeded {
			_ = c.Error(apperror.New(
				http.StatusGatewayTimeout,
				"Hết thời gian yêu cầu",
				"Hệ thống phản hồi quá lâu. Vui lòng thử lại sau.",
			))
		}
	}
}
