package middleware

import (
	"net/http"
	"runtime/debug"
	"smart-wardrobe-be/internal/shared/application/constants/errorcode"
	"smart-wardrobe-be/pkg/logger"
	"smart-wardrobe-be/pkg/utils/errorutils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func GlobalErrorHandler(log logger.Interface, appEnv string) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				rawStack := string(debug.Stack())

				log.Error("CRITICAL PANIC RECOVERED",
					zap.Any("error", r),
					zap.String("stack", rawStack),
				)

				detail := "Đã có lỗi hệ thống xảy ra. Vui lòng thử lại sau hoặc liên hệ quản trị viên."

				if !c.Writer.Written() {
					res := errorcode.ErrorResponse{
						Status: http.StatusInternalServerError,
						Title:  "Lỗi hệ thống",
						Detail: detail,
					}
					// Nếu ở chế độ dev, đẩy thêm stack trace ra API phản hồi
					if appEnv == "development" || gin.Mode() == gin.DebugMode {
						res.StackTrace = errorutils.FilterStackTraceArray(rawStack)
					}
					c.JSON(http.StatusInternalServerError, res)
				}
				c.Abort()
			}
		}()

		c.Next()

		if len(c.Errors) > 0 {
			rawErr := c.Errors.Last().Err

			log.Error("API Execution Error Logged",
				zap.Error(rawErr),
				zap.String("path", c.Request.URL.Path),
			)

			status, title, detail := errorutils.MapErrorToProblem(rawErr)

			if !c.Writer.Written() {
				res := errorcode.ErrorResponse{
					Status: status,
					Title:  title,
					Detail: detail,
				}

				if appEnv == "development" || gin.Mode() == gin.DebugMode {
					res.StackTrace = errorutils.FilterStackTraceArray(string(debug.Stack()))
				}

				c.JSON(status, res)
			}
			c.Abort()
		}
	}
}
