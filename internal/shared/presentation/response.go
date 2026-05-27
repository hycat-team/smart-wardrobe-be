package presentation

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type APIResponse struct {
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
}

type HandlerFuncErr func(*gin.Context) error

func WrapHandler(fn HandlerFuncErr) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := fn(c); err != nil {
			c.Error(err)
			c.Abort()
		}
	}
}

func Success(c *gin.Context, message string, data any) {
	c.JSON(http.StatusOK, APIResponse{
		Message: message,
		Data:    data,
	})
}

func Created(c *gin.Context, message string, data any) {
	c.JSON(http.StatusCreated, APIResponse{
		Message: message,
		Data:    data,
	})
}
