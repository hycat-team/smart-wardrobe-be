package routes

import (
	"net/http"
	"smart-wardrobe-be/internal/config"
	"smart-wardrobe-be/internal/pkg/logger"
	"smart-wardrobe-be/internal/shared/application/constants/apperror"
	shared_msg "smart-wardrobe-be/internal/shared/infrastructure/messaging"
	shared_pres "smart-wardrobe-be/internal/shared/presentation"
	shared_middleware "smart-wardrobe-be/internal/shared/presentation/middleware"
	"time"

	"github.com/gin-gonic/gin"
)

// NewEngine configures the Gin engine and suppresses verbose route registration logs at startup.
func NewEngine(cfg *config.Config, r *AppRouter, log logger.Interface, rateLimit *shared_middleware.RateLimitMiddleware, bodyLimit gin.HandlerFunc) *gin.Engine {
	if cfg.Server.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}
	gin.DebugPrintRouteFunc = func(string, string, string, int) {}

	engine := gin.New()
	shared_pres.ConfigureSSEDeadlines(
		time.Duration(cfg.Server.SSE.ChunkDeadlineSeconds)*time.Second,
		time.Duration(cfg.Server.SSE.AIChatChunkDeadlineSeconds)*time.Second,
	)
	engine.Use(gin.LoggerWithConfig(gin.LoggerConfig{
		Skip: func(c *gin.Context) bool {
			return c.Request.URL.Path == "/api/v1/health" && c.Writer.Status() == http.StatusOK
		},
	}))

	engine.Use(shared_middleware.GlobalErrorHandler(log, cfg.Server.Env))
	engine.Use(shared_middleware.CORSMiddleware(cfg.Server.FrontEndOrigin))
	engine.Use(bodyLimit)
	if cfg.Server.GzipEnabled {
		engine.Use(shared_middleware.GzipMinSizeMiddleware(cfg))
	}
	engine.Use(shared_middleware.GlobalTimeoutMiddleware(time.Duration(cfg.Server.TimeoutSeconds) * time.Second))
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

	swaggerLock := shared_middleware.SwaggerLockMiddleware(cfg)
	swaggerGroup := engine.Group("")
	swaggerGroup.Use(swaggerLock)
	{
		swaggerGroup.Static("/api-docs", "./api/swagger")
		swaggerGroup.StaticFile("/swagger", "./api/swagger/index.html")
		swaggerGroup.GET("/swagger/*any", func(c *gin.Context) {
			c.File("./api/swagger/index.html")
		})
	}

	api := engine.Group("/api/v1")
	{
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"status":    "healthy",
				"time":      time.Now().Format(time.RFC3339),
				"messaging": shared_msg.GetMetrics(),
			})
		})

		r.IdentityRouter.Init(api)
		r.SubscriptionRouter.Init(api)
		r.WardrobeRouter.Init(api)
		r.DashboardRouter.Init(api)
		r.BrandRouter.Init(api)
		r.FashionRouter.Init(api)
		r.AIExecutorRouter.Init(api)
		r.AdminRouter.Init(api)
	}

	return engine
}
