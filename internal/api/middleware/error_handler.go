package middleware

import (
	"net/http"
	"smart-wardrobe-be/internal/shared/application/constants/errorcode"
	"smart-wardrobe-be/pkg/logger"
	"smart-wardrobe-be/pkg/utils/errorutils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func GlobalErrorHandler(log logger.Interface) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				log.Error("CRITICAL PANIC RECOVERED", zap.Any("error", r))
				detail := "Đã có lỗi hệ thống xảy ra. Vui lòng thử lại sau hoặc liên hệ quản trị viên."

				if !c.Writer.Written() {
					c.JSON(http.StatusInternalServerError, errorcode.ErrorResponse{
						Status: http.StatusInternalServerError,
						Title:  "Lỗi hệ thống",
						Detail: detail,
					})
				}
				c.Abort()
			}
		}()

		c.Next()

		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err

			status, title, detail := errorutils.MapErrorToProblem(err)

			if !c.Writer.Written() {
				c.JSON(status, errorcode.ErrorResponse{
					Status: status,
					Title:  title,
					Detail: detail,
				})
			}
			c.Abort()
		}
	}
}
