package routes

import (
	"net/http"
	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/api/middleware"
	"smart-wardrobe-be/internal/shared/application/constants/apperror"
	"smart-wardrobe-be/pkg/logger"
	"time"

	"github.com/gin-gonic/gin"
)

// NewEngine configures the Gin engine and suppresses verbose route registration logs at startup.
func NewEngine(cfg *config.Config, r *AppRouter, log logger.Interface, rateLimit *middleware.RateLimitMiddleware) *gin.Engine {
	if cfg.Server.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}
	gin.DebugPrintRouteFunc = func(string, string, string, int) {}

	engine := gin.New()
	engine.Use(gin.Logger())

	engine.Use(middleware.GlobalErrorHandler(log, cfg.Server.Env))
	engine.Use(middleware.CORSMiddleware(cfg.Server.FrontEndOrigin))
	engine.Use(middleware.GlobalTimeoutMiddleware(time.Duration(cfg.Server.TimeoutSeconds) * time.Second))
	engine.Use(rateLimit.Handle())

	engine.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, apperror.NewNotFound("Đường dẫn API không tồn tại hoặc phương thức không được hỗ trợ."))
	})

	engine.HandleMethodNotAllowed = true
	engine.NoMethod(func(c *gin.Context) {
		c.JSON(http.StatusMethodNotAllowed, apperror.NewError(
			http.StatusMethodNotAllowed,
			"Phương thức không được hỗ trợ",
			"Phương thức HTTP yêu cầu không được hỗ trợ cho đường dẫn này.",
		))
	})

	engine.Static("/api-docs", "./docs")
	engine.StaticFile("/swagger", "./docs/index.html")
	engine.GET("/swagger/*any", func(c *gin.Context) {
		c.File("./docs/index.html")
	})

	api := engine.Group("/api/v1")
	{
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"status": "healthy",
				"time":   time.Now().Format(time.RFC3339),
			})
		})

		r.AuthRouter.Init(api)
		r.AdminRouter.Init(api)
		r.MeRouter.Init(api)
		r.SubscriptionRouter.Init(api)
		r.WardrobeRouter.Init(api)
		r.OutfitRouter.Init(api)
		r.CategoryRouter.Init(api)
		r.CommunityRouter.Init(api)
	}

	return engine
}
