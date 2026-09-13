package middleware

import (
	"net/http"
	"runtime/debug"
	"smart-wardrobe-be/internal/pkg/logger"
	"smart-wardrobe-be/internal/pkg/utils/errorutils"
	"smart-wardrobe-be/internal/shared/application/constants/apperror"

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
					Status:  http.StatusInternalServerError,
					Title:   "Lỗi hệ thống",
					Message: apperror.PanicRecoveryMessage,
				}

				// Debug details are gated strictly on the configured env, not
				// gin.Mode() — staging/unset envs must not leak internals.
				if appEnv == "development" {
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
			appErr = apperror.NewInternalError(apperror.GenericInternalMessage)
		}

		fields := []zap.Field{
			zap.Int("status", appErr.Status),
			zap.String("title", appErr.Title),
			zap.String("message", appErr.Message),
			zap.String("path", c.Request.URL.Path),
		}
		if rawErr != nil && rawErr.Error() != appErr.Message {
			fields = append(fields, zap.String("error", rawErr.Error()))
		}
		if cause := appErr.Unwrap(); cause != nil {
			if cause.Error() != appErr.Message {
				fields = append(fields, zap.String("cause", cause.Error()))
			}
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
		res.DebugMessage = ""
		// Debug details are gated strictly on the configured env, not
		// gin.Mode() — staging/unset envs must not leak internals.
		if appEnv == "development" {
			if rawErr != nil && rawErr.Error() != appErr.Message {
				res.DebugMessage = rawErr.Error()
			}
			res.StackTrace = errorutils.FilterStackTraceArray(appErr.Stack())
		}

		c.JSON(appErr.Status, res)
		c.Errors = nil
		c.Abort()
	}
}
