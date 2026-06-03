package routes

import (
	"net/http"
	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/api/middleware"
	"smart-wardrobe-be/pkg/logger"
	"time"

	"github.com/gin-gonic/gin"
)

func NewEngine(cfg *config.Config, r *AppRouter, log logger.Interface, rateLimit *middleware.RateLimitMiddleware) *gin.Engine {
	engine := gin.Default()

	engine.Use(middleware.GlobalErrorHandler(log, cfg.Server.Env))
	engine.Use(middleware.CORSMiddleware(cfg.Server.FrontEndOrigin))
	engine.Use(middleware.GlobalTimeoutMiddleware(time.Duration(cfg.Server.TimeoutSeconds) * time.Second))
	engine.Use(rateLimit.Handle())

	if cfg.Server.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

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
		r.MeRouter.Init(api)
		r.SubscriptionRouter.Init(api)
		r.WardrobeRouter.Init(api)
		r.OutfitRouter.Init(api)
		r.CategoryRouter.Init(api)
	}

	return engine
}
