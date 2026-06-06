package middleware

import (
	"net/http"
	"runtime/debug"
	"smart-wardrobe-be/internal/shared/application/constants/apperror"
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
					zap.String("source", errorutils.PrimaryStackFrame(rawStack)),
					zap.String("path", c.Request.URL.Path),
				)

				if c.Writer.Written() {
					c.Abort()
					return
				}

				res := apperror.Error{
					Status: http.StatusInternalServerError,
					Title:  "Lỗi hệ thống",
					Detail: "Hệ thống đang gặp sự cố. Vui lòng thử lại sau ít phút hoặc liên hệ quản trị viên.",
				}

				if appEnv == "development" || gin.Mode() == gin.DebugMode {
					res.StackTrace = errorutils.FilterStackTraceArray(rawStack)
				}

				c.JSON(http.StatusInternalServerError, res)
				c.Errors = nil
				c.Abort()
			}
		}()

		c.Next()

		if len(c.Errors) == 0 {
			return
		}

		rawErr := c.Errors.Last().Err
		appErr := errorutils.ToAppError(rawErr)
		if appErr == nil {
			appErr = apperror.NewInternalError("Đã xảy ra lỗi hệ thống. Vui lòng thử lại sau.")
		}

		fields := []zap.Field{
			zap.Int("status", appErr.Status),
			zap.String("title", appErr.Title),
			zap.String("detail", appErr.Detail),
			zap.String("path", c.Request.URL.Path),
		}
		if rawErr != nil && rawErr.Error() != appErr.Detail {
			fields = append(fields, zap.String("error", rawErr.Error()))
		}
		if source := errorutils.PrimaryStackFrame(appErr.Stack()); source != "" {
			fields = append(fields, zap.String("source", source))
		}

		log.Error("API Execution Error Logged", fields...)

		if c.Writer.Written() {
			c.Abort()
			return
		}

		res := *appErr
		res.StackTrace = nil
		if appEnv == "development" || gin.Mode() == gin.DebugMode {
			res.StackTrace = errorutils.FilterStackTraceArray(appErr.Stack())
		}

		c.JSON(appErr.Status, res)
		c.Errors = nil
		c.Abort()
	}
}
